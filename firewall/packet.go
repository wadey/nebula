package firewall

import (
	"encoding/json"
	"fmt"
	mathrand "math/rand"

	"github.com/slackhq/nebula/iputil"
)

type m map[string]interface{}

const (
	ProtoAny  = 0 // When we want to handle HOPOPT (0) we can change this, if ever
	ProtoTCP  = 6
	ProtoUDP  = 17
	ProtoICMP = 1

	PortAny      = 0  // Special value for matching `port: any`
	PortFragment = -1 // Special value for matching `port: fragment`
)

type Packet struct {
	LocalIP    iputil.VpnIp
	RemoteIP   iputil.VpnIp
	LocalPort  uint16
	RemotePort uint16
	Protocol   uint8
	Fragment   bool
}

func (fp *Packet) Copy() *Packet {
	return &Packet{
		LocalIP:    fp.LocalIP,
		RemoteIP:   fp.RemoteIP,
		LocalPort:  fp.LocalPort,
		RemotePort: fp.RemotePort,
		Protocol:   fp.Protocol,
		Fragment:   fp.Fragment,
	}
}

func (fp Packet) MarshalJSON() ([]byte, error) {
	var proto string
	switch fp.Protocol {
	case ProtoTCP:
		proto = "tcp"
	case ProtoICMP:
		proto = "icmp"
	case ProtoUDP:
		proto = "udp"
	default:
		proto = fmt.Sprintf("unknown %v", fp.Protocol)
	}
	return json.Marshal(m{
		"LocalIP":    fp.LocalIP.String(),
		"RemoteIP":   fp.RemoteIP.String(),
		"LocalPort":  fp.LocalPort,
		"RemotePort": fp.RemotePort,
		"Protocol":   proto,
		"Fragment":   fp.Fragment,
	})
}

// UDPSendPort calculates the UDP port to send from when using multiport mode.
// The result will be from [0, numBuckets)
func (fp Packet) UDPSendPort(numBuckets int) uint16 {
	if numBuckets <= 1 {
		return 0
	}

	// If there is no port (like an ICMP packet), pick a random UDP send port
	if fp.LocalPort == 0 {
		return uint16(mathrand.Intn(numBuckets))
	}

	// A decent enough 32bit hash function
	// Prospecting for Hash Functions
	// - https://nullprogram.com/blog/2018/07/31/
	// - https://github.com/skeeto/hash-prospector
	//   [16 21f0aaad 15 d35a2d97 15] = 0.10760229515479501
	x := (uint32(fp.LocalPort) << 16) | uint32(fp.RemotePort)
	x ^= x >> 16
	x *= 0x21f0aaad
	x ^= x >> 15
	x *= 0xd35a2d97
	x ^= x >> 15

	return uint16(x) % uint16(numBuckets)
}
