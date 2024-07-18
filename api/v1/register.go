package api

import (
	"encoding/binary"
	"net"
)

func NewIpv4Addr(ip net.IP, port uint32) *Ipv4Addr {
	ipp := Ipv4Addr{Port: port}
	ipp.Ip = IpToUint32(ip)
	return &ipp
}

func IpToUint32(ip []byte) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func NewIpv6Addr(ip net.IP, port uint32) *Ipv6Addr {
	return &Ipv6Addr{
		Hi:   binary.BigEndian.Uint64(ip[:8]),
		Lo:   binary.BigEndian.Uint64(ip[8:]),
		Port: port,
	}
}
