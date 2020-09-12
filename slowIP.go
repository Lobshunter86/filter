package filter

import (
	"net"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

// IP2Int convert net.IP to uint32
// utilize the impletementation of net.IP for better performance than string2Int version
// it could potentially crash if the impletement is changed
func IP2Int(ip net.IP) uint32 {
	// for ipv4, the last 4 bytes are used to store IP address
	ip4 := ip[len(ip)-4:]
	var u uint32 = 0

	// u |= uint32(ip4[3])
	// u |= uint32(ip4[2]) << 8
	// u |= uint32(ip4[1]) << 16
	// u |= uint32(ip4[0]) << 24

	// friendly to CPU parallel computing
	u |= (uint32(ip4[3]) | uint32(ip4[2])<<8) |
		(uint32(ip4[1])<<16 | uint32(ip4[0])<<24)

	return u
}
