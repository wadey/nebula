package nebula

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/cert"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/test"
	"github.com/slackhq/nebula/udp"
	"github.com/stretchr/testify/assert"
)

func TestControl_GetHostInfoByVpnIp(t *testing.T) {
	l := test.NewLogger()
	// Special care must be taken to re-use all objects provided to the hostmap and certificate in the expectedInfo object
	// To properly ensure we are not exposing core memory to the caller
	hm := NewHostMap(l, "test", &net.IPNet{}, make([]*net.IPNet, 0))
	remote1 := udp.NewAddr(net.ParseIP("0.0.0.100"), 4444)
	remote2 := udp.NewAddr(net.ParseIP("1:2:3:4:5:6:7:8"), 4444)
	ipNet := net.IPNet{
		IP:   net.IPv4(1, 2, 3, 4),
		Mask: net.IPMask{255, 255, 255, 0},
	}

	ipNet2 := net.IPNet{
		IP:   net.ParseIP("1:2:3:4:5:6:7:8"),
		Mask: net.IPMask{255, 255, 255, 0},
	}

	crt := &cert.NebulaCertificate{
		Details: cert.NebulaCertificateDetails{
			Name:           "test",
			Ips:            []*net.IPNet{&ipNet},
			Subnets:        []*net.IPNet{},
			Groups:         []string{"default-group"},
			NotBefore:      time.Unix(1, 0),
			NotAfter:       time.Unix(2, 0),
			PublicKey:      []byte{5, 6, 7, 8},
			IsCA:           false,
			Issuer:         "the-issuer",
			InvertedGroups: map[string]struct{}{"default-group": {}},
		},
		Signature: []byte{1, 2, 1, 2, 1, 3},
	}

	remotes := NewRemoteList()
	remotes.unlockedPrependV4(0, NewIp4AndPort(remote1.IP, uint32(remote1.Port)))
	remotes.unlockedPrependV6(0, NewIp6AndPort(remote2.IP, uint32(remote2.Port)))
	hm.Add(iputil.Ip2VpnIp(ipNet.IP), &HostInfo{
		remote:  remote1,
		remotes: remotes,
		ConnectionState: &ConnectionState{
			peerCert: crt,
		},
		remoteIndexId: 200,
		localIndexId:  201,
		vpnIp:         iputil.Ip2VpnIp(ipNet.IP),
	})

	hm.Add(iputil.Ip2VpnIp(ipNet2.IP), &HostInfo{
		remote:  remote1,
		remotes: remotes,
		ConnectionState: &ConnectionState{
			peerCert: nil,
		},
		remoteIndexId: 200,
		localIndexId:  201,
		vpnIp:         iputil.Ip2VpnIp(ipNet2.IP),
	})

	c := Control{
		f: &Interface{
			hostMap: hm,
		},
		l: logrus.New(),
	}

	thi := c.GetHostInfoByVpnIp(iputil.Ip2VpnIp(ipNet.IP), false)

	expectedInfo := ControlHostInfo{
		VpnIp:          net.IPv4(1, 2, 3, 4).To4(),
		LocalIndex:     201,
		RemoteIndex:    200,
		RemoteAddrs:    []*udp.Addr{remote2, remote1},
		CachedPackets:  0,
		Cert:           crt.Copy(),
		MessageCounter: 0,
		CurrentRemote:  udp.NewAddr(net.ParseIP("0.0.0.100"), 4444),
	}

	// Make sure we don't have any unexpected fields
	assertFields(t, []string{"VpnIp", "LocalIndex", "RemoteIndex", "RemoteAddrs", "CachedPackets", "Cert", "MessageCounter", "CurrentRemote"}, thi)
	test.AssertDeepCopyEqual(t, &expectedInfo, thi)

	// Make sure we don't panic if the host info doesn't have a cert yet
	assert.NotPanics(t, func() {
		thi = c.GetHostInfoByVpnIp(iputil.Ip2VpnIp(ipNet2.IP), false)
	})
}

func assertFields(t *testing.T, expected []string, actualStruct interface{}) {
	val := reflect.ValueOf(actualStruct).Elem()
	fields := make([]string, val.NumField())
	for i := 0; i < val.NumField(); i++ {
		fields[i] = val.Type().Field(i).Name
	}

	assert.Equal(t, expected, fields)
}
