package nebula

import (
	"net"
	"testing"

	"github.com/slackhq/nebula/test"
	"github.com/stretchr/testify/assert"
)

func TestHostMap_MakePrimary(t *testing.T) {
	l := test.NewLogger()
	hm := NewHostMap(
		l, "test",
		&net.IPNet{
			IP:   net.IP{10, 0, 0, 1},
			Mask: net.IPMask{255, 255, 255, 0},
		},
		[]*net.IPNet{},
	)

	f := &Interface{}

	h1 := &HostInfo{vpnIp: 1, localIndexId: 1}
	h2 := &HostInfo{vpnIp: 1, localIndexId: 2}
	h3 := &HostInfo{vpnIp: 1, localIndexId: 3}
	h4 := &HostInfo{vpnIp: 1, localIndexId: 4}

	hm.unlockedAddHostInfo(h4, f)
	hm.unlockedAddHostInfo(h3, f)
	hm.unlockedAddHostInfo(h2, f)
	hm.unlockedAddHostInfo(h1, f)

	// Make sure we go h1 -> h2 -> h3 -> h4
	prim, _ := hm.QueryVpnIp(1)
	assert.Equal(t, h1.localIndexId, prim.localIndexId)
	assert.Equal(t, h2.localIndexId, prim.next.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Equal(t, h1.localIndexId, h2.prev.localIndexId)
	assert.Equal(t, h3.localIndexId, h2.next.localIndexId)
	assert.Equal(t, h2.localIndexId, h3.prev.localIndexId)
	assert.Equal(t, h4.localIndexId, h3.next.localIndexId)
	assert.Equal(t, h3.localIndexId, h4.prev.localIndexId)
	assert.Nil(t, h4.next)

	// Swap h3/middle to primary
	hm.MakePrimary(h3)

	// Make sure we go h3 -> h1 -> h2 -> h4
	prim, _ = hm.QueryVpnIp(1)
	assert.Equal(t, h3.localIndexId, prim.localIndexId)
	assert.Equal(t, h1.localIndexId, prim.next.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Equal(t, h2.localIndexId, h1.next.localIndexId)
	assert.Equal(t, h3.localIndexId, h1.prev.localIndexId)
	assert.Equal(t, h4.localIndexId, h2.next.localIndexId)
	assert.Equal(t, h1.localIndexId, h2.prev.localIndexId)
	assert.Equal(t, h2.localIndexId, h4.prev.localIndexId)
	assert.Nil(t, h4.next)

	// Swap h4/tail to primary
	hm.MakePrimary(h4)

	// Make sure we go h4 -> h3 -> h1 -> h2
	prim, _ = hm.QueryVpnIp(1)
	assert.Equal(t, h4.localIndexId, prim.localIndexId)
	assert.Equal(t, h3.localIndexId, prim.next.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Equal(t, h1.localIndexId, h3.next.localIndexId)
	assert.Equal(t, h4.localIndexId, h3.prev.localIndexId)
	assert.Equal(t, h2.localIndexId, h1.next.localIndexId)
	assert.Equal(t, h3.localIndexId, h1.prev.localIndexId)
	assert.Equal(t, h1.localIndexId, h2.prev.localIndexId)
	assert.Nil(t, h2.next)

	// Swap h4 again should be no-op
	hm.MakePrimary(h4)

	// Make sure we go h4 -> h3 -> h1 -> h2
	prim, _ = hm.QueryVpnIp(1)
	assert.Equal(t, h4.localIndexId, prim.localIndexId)
	assert.Equal(t, h3.localIndexId, prim.next.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Equal(t, h1.localIndexId, h3.next.localIndexId)
	assert.Equal(t, h4.localIndexId, h3.prev.localIndexId)
	assert.Equal(t, h2.localIndexId, h1.next.localIndexId)
	assert.Equal(t, h3.localIndexId, h1.prev.localIndexId)
	assert.Equal(t, h1.localIndexId, h2.prev.localIndexId)
	assert.Nil(t, h2.next)
}

func TestHostMap_DeleteHostInfo(t *testing.T) {
	l := test.NewLogger()
	hm := NewHostMap(
		l, "test",
		&net.IPNet{
			IP:   net.IP{10, 0, 0, 1},
			Mask: net.IPMask{255, 255, 255, 0},
		},
		[]*net.IPNet{},
	)

	f := &Interface{}

	h1 := &HostInfo{vpnIp: 1, localIndexId: 1}
	h2 := &HostInfo{vpnIp: 1, localIndexId: 2}
	h3 := &HostInfo{vpnIp: 1, localIndexId: 3}
	h4 := &HostInfo{vpnIp: 1, localIndexId: 4}
	h5 := &HostInfo{vpnIp: 1, localIndexId: 5}
	h6 := &HostInfo{vpnIp: 1, localIndexId: 6}

	hm.unlockedAddHostInfo(h6, f)
	hm.unlockedAddHostInfo(h5, f)
	hm.unlockedAddHostInfo(h4, f)
	hm.unlockedAddHostInfo(h3, f)
	hm.unlockedAddHostInfo(h2, f)
	hm.unlockedAddHostInfo(h1, f)

	// h6 should be deleted
	assert.Nil(t, h6.next)
	assert.Nil(t, h6.prev)
	_, err := hm.QueryIndex(h6.localIndexId)
	assert.Error(t, err)

	// Make sure we go h1 -> h2 -> h3 -> h4 -> h5
	prim, _ := hm.QueryVpnIp(1)
	assert.Equal(t, h1.localIndexId, prim.localIndexId)
	assert.Equal(t, h2.localIndexId, prim.next.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Equal(t, h1.localIndexId, h2.prev.localIndexId)
	assert.Equal(t, h3.localIndexId, h2.next.localIndexId)
	assert.Equal(t, h2.localIndexId, h3.prev.localIndexId)
	assert.Equal(t, h4.localIndexId, h3.next.localIndexId)
	assert.Equal(t, h3.localIndexId, h4.prev.localIndexId)
	assert.Equal(t, h5.localIndexId, h4.next.localIndexId)
	assert.Equal(t, h4.localIndexId, h5.prev.localIndexId)
	assert.Nil(t, h5.next)

	// Delete primary
	hm.DeleteHostInfo(h1)
	assert.Nil(t, h1.prev)
	assert.Nil(t, h1.next)

	// Make sure we go h2 -> h3 -> h4 -> h5
	prim, _ = hm.QueryVpnIp(1)
	assert.Equal(t, h2.localIndexId, prim.localIndexId)
	assert.Equal(t, h3.localIndexId, prim.next.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Equal(t, h3.localIndexId, h2.next.localIndexId)
	assert.Equal(t, h2.localIndexId, h3.prev.localIndexId)
	assert.Equal(t, h4.localIndexId, h3.next.localIndexId)
	assert.Equal(t, h3.localIndexId, h4.prev.localIndexId)
	assert.Equal(t, h5.localIndexId, h4.next.localIndexId)
	assert.Equal(t, h4.localIndexId, h5.prev.localIndexId)
	assert.Nil(t, h5.next)

	// Delete in the middle
	hm.DeleteHostInfo(h3)
	assert.Nil(t, h3.prev)
	assert.Nil(t, h3.next)

	// Make sure we go h2 -> h4 -> h5
	prim, _ = hm.QueryVpnIp(1)
	assert.Equal(t, h2.localIndexId, prim.localIndexId)
	assert.Equal(t, h4.localIndexId, prim.next.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Equal(t, h4.localIndexId, h2.next.localIndexId)
	assert.Equal(t, h2.localIndexId, h4.prev.localIndexId)
	assert.Equal(t, h5.localIndexId, h4.next.localIndexId)
	assert.Equal(t, h4.localIndexId, h5.prev.localIndexId)
	assert.Nil(t, h5.next)

	// Delete the tail
	hm.DeleteHostInfo(h5)
	assert.Nil(t, h5.prev)
	assert.Nil(t, h5.next)

	// Make sure we go h2 -> h4
	prim, _ = hm.QueryVpnIp(1)
	assert.Equal(t, h2.localIndexId, prim.localIndexId)
	assert.Equal(t, h4.localIndexId, prim.next.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Equal(t, h4.localIndexId, h2.next.localIndexId)
	assert.Equal(t, h2.localIndexId, h4.prev.localIndexId)
	assert.Nil(t, h4.next)

	// Delete the head
	hm.DeleteHostInfo(h2)
	assert.Nil(t, h2.prev)
	assert.Nil(t, h2.next)

	// Make sure we only have h4
	prim, _ = hm.QueryVpnIp(1)
	assert.Equal(t, h4.localIndexId, prim.localIndexId)
	assert.Nil(t, prim.prev)
	assert.Nil(t, prim.next)
	assert.Nil(t, h4.next)

	// Delete the only item
	hm.DeleteHostInfo(h4)
	assert.Nil(t, h4.prev)
	assert.Nil(t, h4.next)

	// Make sure we have nil
	prim, _ = hm.QueryVpnIp(1)
	assert.Nil(t, prim)
}
