package nebula

import (
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/flynn/noise"
	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/cert"
	"github.com/stretchr/testify/require"
)

func BenchmarkInsideHotPath(b *testing.B) {
	l.SetLevel(logrus.WarnLevel)
	header, _ := hex.DecodeString(
		// IP packet, 192.168.0.120 -> 192.168.0.1
		// UDP packet, port 52228 -> 9999
		// body: all zeros, total length 1500
		"450005dc75ad400040113d9ac0a80078c0a80001" + "cc04270f05c87f80",
	)

	packet := make([]byte, mtu)
	copy(packet[0:], header)

	fwPacket := &FirewallPacket{}

	out := make([]byte, mtu)
	nb := make([]byte, 12, 12)

	myIp, myNet, _ := net.ParseCIDR("192.168.0.120/24")
	myIpNet := &net.IPNet{
		IP:   myIp,
		Mask: myNet.Mask,
	}
	_, localToMe, _ := net.ParseCIDR("10.0.0.1/8")
	myIpNets := []*net.IPNet{myIpNet}
	preferredRanges := []*net.IPNet{localToMe}

	c := cert.NebulaCertificate{
		Details: cert.NebulaCertificateDetails{
			Name:           "host1",
			Ips:            myIpNets,
			InvertedGroups: map[string]struct{}{"default-group": {}, "test-group": {}},
		},
	}

	fw := NewFirewall(time.Second, time.Minute, time.Hour, &c)
	require.NoError(b, fw.AddRule(false, fwProtoAny, 0, 0, []string{"any"}, "", nil, "", ""))

	hostMap := NewHostMap("main", myIpNet, preferredRanges)
	// TODO should we send to port 9 (discard protocol) instead of ourselves?
	// Sending to :9 seems to slow down the test since another service on the
	// box has to recv the messages. If we just send to ourselves, the packets
	// just fill the buffer and get thrown away.
	hostMap.AddRemote(ip2int(net.ParseIP("192.168.0.1")), NewUDPAddrFromString("127.0.0.1:4242"))
	info, _ := hostMap.QueryVpnIP(ip2int(net.ParseIP("192.168.0.1")))
	var mc uint64
	info.ConnectionState = &ConnectionState{
		ready:          true,
		messageCounter: &mc,
	}
	info.HandshakeReady = true

	ifce := &Interface{
		hostMap:    hostMap,
		firewall:   fw,
		lightHouse: &LightHouse{},
		outside:    &dropOutside{},
	}
	ifce.connectionManager = newConnectionManager(ifce, 300, 300)

	packet = packet[:1500]

	b.Run("AESGCM", func(b *testing.B) {
		info.ConnectionState.eKey = testHotPathCipherState(b, noise.CipherAESGCM)

		// Prep the hot path, add to conntrack
		ifce.consumeInsidePacket(packet, fwPacket, nb, out)

		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			ifce.consumeInsidePacket(packet, fwPacket, nb, out)
		}
		b.SetBytes(1500)
	})
	b.Run("ChaChaPoly", func(b *testing.B) {
		info.ConnectionState.eKey = testHotPathCipherState(b, noise.CipherChaChaPoly)

		// Prep the hot path, add to conntrack
		ifce.consumeInsidePacket(packet, fwPacket, nb, out)

		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			ifce.consumeInsidePacket(packet, fwPacket, nb, out)
		}
		b.SetBytes(1500)
	})
}

// Drop all outgoing packets, for Benchmark test
type dropOutside struct{}

func (dropOutside) WriteTo(b []byte, addr *udpAddr) error { return nil }
func (dropOutside) LocalAddr() (*udpAddr, error)          { return nil, nil }
func (dropOutside) ListenOut(f *Interface)                {}
func (dropOutside) reloadConfig(c *Config)                {}
