package cidr

import (
	"net"

	"github.com/slackhq/nebula/iputil"
)

const startbit6 = uint64(1 << 63)

type Tree6 struct {
	root4 *Node
	root6 *Node
}

func NewTree6() *Tree6 {
	tree := new(Tree6)
	tree.root4 = &Node{}
	tree.root6 = &Node{}
	return tree
}

func (tree *Tree6) AddCIDR(cidr *net.IPNet, val interface{}) {
	var node, next *Node

	cidrIP, ipv4 := isIPV4(cidr.IP)
	if ipv4 {
		node = tree.root4
		next = tree.root4

	} else {
		node = tree.root6
		next = tree.root6
	}

	for i := 0; i < len(cidrIP); i += 4 {
		ip := iputil.Ip2VpnIp(cidrIP[i : i+4])
		mask := iputil.Ip2VpnIp(cidr.Mask[i : i+4])
		bit := startbit

		// Find our last ancestor in the tree
		for bit&mask != 0 {
			if ip&bit != 0 {
				next = node.right
			} else {
				next = node.left
			}

			if next == nil {
				break
			}

			bit = bit >> 1
			node = next
		}

		// Build up the rest of the tree we don't already have
		for bit&mask != 0 {
			next = &Node{}
			next.parent = node

			if ip&bit != 0 {
				node.right = next
			} else {
				node.left = next
			}

			bit >>= 1
			node = next
		}
	}

	// Final node marks our cidr, set the value
	node.value = val
}

// Finds the most specific match
func (tree *Tree6) MostSpecificContains(ip net.IP) (value interface{}) {
	var node *Node

	wholeIP, ipv4 := isIPV4(ip)
	if ipv4 {
		node = tree.root4
	} else {
		node = tree.root6
	}

	for i := 0; i < len(wholeIP); i += 4 {
		ip := iputil.Ip2VpnIp(wholeIP[i : i+4])
		bit := startbit

		for node != nil {
			if node.value != nil {
				value = node.value
			}

			if bit == 0 {
				break
			}

			if ip&bit != 0 {
				node = node.right
			} else {
				node = node.left
			}

			bit >>= 1
		}
	}

	return value
}

func (tree *Tree6) MostSpecificContainsIpV4(ip iputil.VpnIp) (value interface{}) {
	bit := startbit
	node := tree.root4

	for node != nil {
		if node.value != nil {
			value = node.value
		}

		if ip&bit != 0 {
			node = node.right
		} else {
			node = node.left
		}

		bit >>= 1
	}

	return value
}

func (tree *Tree6) MostSpecificContainsIpV6(hi, lo uint64) (value interface{}) {
	ip := hi
	node := tree.root6

	for i := 0; i < 2; i++ {
		bit := startbit6

		for node != nil {
			if node.value != nil {
				value = node.value
			}

			if bit == 0 {
				break
			}

			if ip&bit != 0 {
				node = node.right
			} else {
				node = node.left
			}

			bit >>= 1
		}

		ip = lo
	}

	return value
}

func isIPV4(ip net.IP) (net.IP, bool) {
	if len(ip) == net.IPv4len {
		return ip, true
	}

	if len(ip) == net.IPv6len && isZeros(ip[0:10]) && ip[10] == 0xff && ip[11] == 0xff {
		return ip[12:16], true
	}

	return ip, false
}

func isZeros(p net.IP) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != 0 {
			return false
		}
	}
	return true
}
