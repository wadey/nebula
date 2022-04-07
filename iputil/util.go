package iputil

import (
	"encoding/binary"
	"fmt"
	"net"
)

type VpnIp uint32

const maxIPv4StringLen = len("255.255.255.255")

func (ip VpnIp) String() string {
	b := make([]byte, maxIPv4StringLen)

	n := ubtoa(b, 0, byte(ip>>24))
	b[n] = '.'
	n++

	n += ubtoa(b, n, byte(ip>>16&255))
	b[n] = '.'
	n++

	n += ubtoa(b, n, byte(ip>>8&255))
	b[n] = '.'
	n++

	n += ubtoa(b, n, byte(ip&255))
	return string(b[:n])
}

func (ip VpnIp) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", ip.String())), nil
}

func (ip VpnIp) ToIP() net.IP {
	nip := make(net.IP, 4)
	binary.BigEndian.PutUint32(nip, uint32(ip))
	return nip
}

func Ip2VpnIp(ip []byte) VpnIp {
	if len(ip) == 16 {
		return VpnIp(binary.BigEndian.Uint32(ip[12:16]))
	}
	return VpnIp(binary.BigEndian.Uint32(ip))
}

// ubtoa encodes the string form of the integer v to dst[start:] and
// returns the number of bytes written to dst. The caller must ensure
// that dst has sufficient length.
func ubtoa(dst []byte, start int, v byte) int {
	if v < 10 {
		dst[start] = v + '0'
		return 1
	} else if v < 100 {
		dst[start+1] = v%10 + '0'
		dst[start] = v/10 + '0'
		return 2
	}

	dst[start+2] = v%10 + '0'
	dst[start+1] = (v/10)%10 + '0'
	dst[start] = v/100 + '0'
	return 3
}
