package nebula

import (
	"net"
	"testing"
	"time"

	"github.com/flynn/noise"
	"github.com/slackhq/nebula/cert"
	"github.com/stretchr/testify/assert"
)

var vpnIP uint32

func Test_NewConnectionManagerTest(t *testing.T) {
	//_, tuncidr, _ := net.ParseCIDR("1.1.1.1/24")
	_, vpncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, localrange, _ := net.ParseCIDR("10.1.1.1/24")
	vpnIP = ip2int(net.ParseIP("172.1.1.2"))
	preferredRanges := []*net.IPNet{localrange}

	// Very incomplete mock objects
	hostMap := NewHostMap("test", vpncidr, preferredRanges)
	cs := &CertState{
		rawCertificate:      []byte{},
		privateKey:          []byte{},
		certificate:         &cert.NebulaCertificate{},
		rawCertificateNoKey: []byte{},
	}

	lh := NewLightHouse(false, 0, []uint32{}, 1000, 0, &udpConn{}, false)
	ifce := &Interface{
		hostMap:          hostMap,
		inside:           &Tun{},
		outside:          &udpConn{},
		certState:        cs,
		firewall:         &Firewall{},
		lightHouse:       lh,
		handshakeManager: NewHandshakeManager(vpncidr, preferredRanges, hostMap, lh, &udpConn{}, defaultHandshakeConfig),
	}
	now := time.Now()

	// Create manager
	nc := newConnectionManager(ifce, 5, 10)
	nc.HandleMonitorTick(now)
	// Add an ip we have established a connection w/ to hostmap
	hostinfo := nc.hostMap.AddVpnIP(vpnIP)
	hostinfo.ConnectionState = &ConnectionState{
		certState:      cs,
		H:              &noise.HandshakeState{},
		messageCounter: new(uint64),
	}

	// We saw traffic out to vpnIP
	nc.Out(vpnIP)
	assert.NotContains(t, nc.pendingDeletion, vpnIP)
	assert.Contains(t, nc.hostMap.Hosts, vpnIP)
	// Move ahead 5s. Nothing should happen
	next_tick := now.Add(5 * time.Second)
	nc.HandleMonitorTick(next_tick)
	nc.HandleDeletionTick(next_tick)
	// Move ahead 6s. We haven't heard back
	next_tick = now.Add(6 * time.Second)
	nc.HandleMonitorTick(next_tick)
	nc.HandleDeletionTick(next_tick)
	// This host should now be up for deletion
	assert.Contains(t, nc.pendingDeletion, vpnIP)
	assert.Contains(t, nc.hostMap.Hosts, vpnIP)
	// Move ahead some more
	next_tick = now.Add(45 * time.Second)
	nc.HandleMonitorTick(next_tick)
	nc.HandleDeletionTick(next_tick)
	// The host should be evicted
	assert.NotContains(t, nc.pendingDeletion, vpnIP)
	assert.NotContains(t, nc.hostMap.Hosts, vpnIP)

}

func Test_NewConnectionManagerTest2(t *testing.T) {
	//_, tuncidr, _ := net.ParseCIDR("1.1.1.1/24")
	_, vpncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, localrange, _ := net.ParseCIDR("10.1.1.1/24")
	preferredRanges := []*net.IPNet{localrange}

	// Very incomplete mock objects
	hostMap := NewHostMap("test", vpncidr, preferredRanges)
	cs := &CertState{
		rawCertificate:      []byte{},
		privateKey:          []byte{},
		certificate:         &cert.NebulaCertificate{},
		rawCertificateNoKey: []byte{},
	}

	lh := NewLightHouse(false, 0, []uint32{}, 1000, 0, &udpConn{}, false)
	ifce := &Interface{
		hostMap:          hostMap,
		inside:           &Tun{},
		outside:          &udpConn{},
		certState:        cs,
		firewall:         &Firewall{},
		lightHouse:       lh,
		handshakeManager: NewHandshakeManager(vpncidr, preferredRanges, hostMap, lh, &udpConn{}, defaultHandshakeConfig),
	}
	now := time.Now()

	// Create manager
	nc := newConnectionManager(ifce, 5, 10)
	nc.HandleMonitorTick(now)
	// Add an ip we have established a connection w/ to hostmap
	hostinfo := nc.hostMap.AddVpnIP(vpnIP)
	hostinfo.ConnectionState = &ConnectionState{
		certState:      cs,
		H:              &noise.HandshakeState{},
		messageCounter: new(uint64),
	}

	// We saw traffic out to vpnIP
	nc.Out(vpnIP)
	assert.NotContains(t, nc.pendingDeletion, vpnIP)
	assert.Contains(t, nc.hostMap.Hosts, vpnIP)
	// Move ahead 5s. Nothing should happen
	next_tick := now.Add(5 * time.Second)
	nc.HandleMonitorTick(next_tick)
	nc.HandleDeletionTick(next_tick)
	// Move ahead 6s. We haven't heard back
	next_tick = now.Add(6 * time.Second)
	nc.HandleMonitorTick(next_tick)
	nc.HandleDeletionTick(next_tick)
	// This host should now be up for deletion
	assert.Contains(t, nc.pendingDeletion, vpnIP)
	assert.Contains(t, nc.hostMap.Hosts, vpnIP)
	// We heard back this time
	nc.In(vpnIP)
	// Move ahead some more
	next_tick = now.Add(45 * time.Second)
	nc.HandleMonitorTick(next_tick)
	nc.HandleDeletionTick(next_tick)
	// The host should be evicted
	assert.NotContains(t, nc.pendingDeletion, vpnIP)
	assert.Contains(t, nc.hostMap.Hosts, vpnIP)

}
