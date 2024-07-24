package utils

import (
	"encoding/binary"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"net"
	"net/netip"
)

func NewUDPConn(localAddr string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func LocalIps() *[]net.IP {
	//FIXME: This function is pretty garbage
	var ips []net.IP
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				//continue
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// TODO: 暂时过滤掉链路本地地址，这是最正确的做法
			// TODO: 也希望能过滤掉基于 SLAAC MAC 地址的 IP
			if ip.IsLoopback() == false && !ip.IsLinkLocalUnicast() {
				ips = append(ips, ip)
			}
		}
	}
	return &ips
}

func lhIp6ToIp(v *api.Ipv6Addr) net.IP {
	ip := make(net.IP, 16)
	binary.BigEndian.PutUint64(ip[:8], v.Hi)
	binary.BigEndian.PutUint64(ip[8:], v.Lo)
	return ip
}

func NewUDPAddrFromLH6(ipp *api.Ipv6Addr) *udp.Addr {
	return udp.NewAddr(lhIp6ToIp(ipp), uint16(ipp.Port))
}

func NewIp6AndPortFromNetIP(ip netip.Addr, port uint16) *api.Ipv6Addr {
	ip6Addr := ip.As16()
	return &api.Ipv6Addr{
		Hi:   binary.BigEndian.Uint64(ip6Addr[:8]),
		Lo:   binary.BigEndian.Uint64(ip6Addr[8:]),
		Port: uint32(port),
	}
}

func NewUDPAddrFromLH4(ipp *api.Ipv4Addr) *udp.Addr {
	ip := ipp.Ip
	return udp.NewAddr(
		net.IPv4(byte(ip&0xff000000>>24), byte(ip&0x00ff0000>>16), byte(ip&0x0000ff00>>8), byte(ip&0x000000ff)),
		uint16(ipp.Port),
	)
}
