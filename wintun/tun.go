//go:build windows
// +build windows

/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

//NOTE: This file was forked from https://git.zx2c4.com/wireguard-go/tree/tun/tun_windows.go?id=851efb1bb65555e0f765a3361c8eb5ac47435b19
//  and https://git.zx2c4.com/wireguard-go/tree/tun/tun_windows.go?id=0ad14a89f5f9da577dae6a63ad196015e51a0381
// Mainly to shed functionality we won't be using and to fix names that display in the system

package wintun

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
	_ "unsafe"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wintun"
)

const (
	rateMeasurementGranularity = uint64((time.Second / 2) / time.Nanosecond)
	spinloopRateThreshold      = 800000000 / 8                                   // 800mbps
	spinloopDuration           = uint64(time.Millisecond / 80 / time.Nanosecond) // ~1gbit/s
)

type rateJuggler struct {
	current       atomic.Uint64
	nextByteCount atomic.Uint64
	nextStartTime atomic.Int64
	changing      atomic.Bool
}

type NativeTun struct {
	wt        *wintun.Adapter
	name      string
	handle    windows.Handle
	rate      rateJuggler
	session   wintun.Session
	readWait  windows.Handle
	running   sync.WaitGroup
	closeOnce sync.Once
	close     atomic.Bool
}

var (
	WintunTunnelType          = "Nebula"
	WintunStaticRequestedGUID *windows.GUID
)

//go:linkname procyield runtime.procyield
func procyield(cycles uint32)

//go:linkname nanotime runtime.nanotime
func nanotime() int64

// CreateTUN creates a Wintun interface with the given name. Should a Wintun
// interface with the same name exist, it is reused.
func CreateTUN(ifname string, mtu int) (Device, error) {
	return CreateTUNWithRequestedGUID(ifname, WintunStaticRequestedGUID, mtu)
}

// CreateTUNWithRequestedGUID creates a Wintun interface with the given name and
// a requested GUID. Should a Wintun interface with the same name exist, it is reused.
func CreateTUNWithRequestedGUID(ifname string, requestedGUID *windows.GUID, mtu int) (Device, error) {
	wt, err := wintun.CreateAdapter(ifname, WintunTunnelType, requestedGUID)
	if err != nil {
		return nil, fmt.Errorf("Error creating interface: %w", err)
	}

	tun := &NativeTun{
		wt:     wt,
		name:   ifname,
		handle: windows.InvalidHandle,
	}

	tun.session, err = wt.StartSession(0x800000) // Ring capacity, 8 MiB
	if err != nil {
		tun.wt.Close()
		return nil, fmt.Errorf("Error starting session: %w", err)
	}
	tun.readWait = tun.session.ReadWaitEvent()
	return tun, nil
}

func (tun *NativeTun) Name() (string, error) {
	return tun.name, nil
}

func (tun *NativeTun) File() *os.File {
	return nil
}

func (tun *NativeTun) Close() error {
	var err error
	tun.closeOnce.Do(func() {
		tun.close.Store(true)
		windows.SetEvent(tun.readWait)
		tun.running.Wait()
		tun.session.End()
		if tun.wt != nil {
			tun.wt.Close()
		}
	})
	return err
}

// Note: Read() and Write() assume the caller comes only from a single thread; there's no locking.

func (tun *NativeTun) Read(buff []byte, offset int) (int, error) {
	tun.running.Add(1)
	defer tun.running.Done()
retry:
	if tun.close.Load() {
		return 0, os.ErrClosed
	}
	start := nanotime()
	shouldSpin := tun.rate.current.Load() >= spinloopRateThreshold && uint64(start-tun.rate.nextStartTime.Load()) <= rateMeasurementGranularity*2
	for {
		if tun.close.Load() {
			return 0, os.ErrClosed
		}
		packet, err := tun.session.ReceivePacket()
		switch err {
		case nil:
			packetSize := len(packet)
			copy(buff[offset:], packet)
			tun.session.ReleaseReceivePacket(packet)
			tun.rate.update(uint64(packetSize))
			return packetSize, nil
		case windows.ERROR_NO_MORE_ITEMS:
			if !shouldSpin || uint64(nanotime()-start) >= spinloopDuration {
				windows.WaitForSingleObject(tun.readWait, windows.INFINITE)
				goto retry
			}
			procyield(1)
			continue
		case windows.ERROR_HANDLE_EOF:
			return 0, os.ErrClosed
		case windows.ERROR_INVALID_DATA:
			return 0, errors.New("Send ring corrupt")
		}
		return 0, fmt.Errorf("Read failed: %w", err)
	}
}

func (tun *NativeTun) Flush() error {
	return nil
}

func (tun *NativeTun) Write(buff []byte, offset int) (int, error) {
	tun.running.Add(1)
	defer tun.running.Done()
	if tun.close.Load() {
		return 0, os.ErrClosed
	}

	packetSize := len(buff) - offset
	tun.rate.update(uint64(packetSize))

	packet, err := tun.session.AllocateSendPacket(packetSize)
	if err == nil {
		copy(packet, buff[offset:])
		tun.session.SendPacket(packet)
		return packetSize, nil
	}
	switch err {
	case windows.ERROR_HANDLE_EOF:
		return 0, os.ErrClosed
	case windows.ERROR_BUFFER_OVERFLOW:
		return 0, nil // Dropping when ring is full.
	}
	return 0, fmt.Errorf("Write failed: %w", err)
}

// LUID returns Windows interface instance ID.
func (tun *NativeTun) LUID() uint64 {
	tun.running.Add(1)
	defer tun.running.Done()
	if tun.close.Load() {
		return 0
	}
	return tun.wt.LUID()
}

// RunningVersion returns the running version of the Wintun driver.
func (tun *NativeTun) RunningVersion() (version uint32, err error) {
	return wintun.RunningVersion()
}

func (rate *rateJuggler) update(packetLen uint64) {
	now := nanotime()
	total := rate.nextByteCount.Add(packetLen)
	period := uint64(now - rate.nextStartTime.Load())
	if period >= rateMeasurementGranularity {
		if !rate.changing.CompareAndSwap(false, true) {
			return
		}
		rate.nextStartTime.Store(now)
		rate.current.Store(total * uint64(time.Second/time.Nanosecond) / period)
		rate.nextByteCount.Store(0)
		rate.changing.Store(false)
	}
}
