package udp

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
)

type Addr struct {
	IP   net.IP
	Port uint16
}

func NewAddr(ip net.IP, port uint16) *Addr {
	addr := Addr{IP: make([]byte, net.IPv6len), Port: port}
	copy(addr.IP, ip.To16())
	return &addr
}

func (a *Addr) Network() string {
	return "udp"
}

func (a *Addr) String() string {
	return fmt.Sprintf("%s:%d", a.IP.String(), a.Port)
}

func (a *Addr) NetAddr() net.Addr {
	return &net.UDPAddr{
		IP:   a.IP,
		Port: int(a.Port),
	}
}

func (a *Addr) Copy() *Addr {
	if a == nil {
		return nil
	}

	nu := Addr{
		Port: a.Port,
		IP:   make(net.IP, len(a.IP)),
	}

	copy(nu.IP, a.IP)
	return &nu
}

func (a *Addr) ToBytesManual() ([]byte, error) {
	ipBytes := a.IP.To4()
	if ipBytes == nil {
		ipBytes = a.IP.To16()
		if ipBytes == nil {
			return nil, fmt.Errorf("invalid IP address")
		}
	}
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, a.Port)
	return append(ipBytes, portBytes...), nil
}

func FromBytesManual(data []byte) (Addr, error) {
	if len(data) != 6 && len(data) != 18 {
		return Addr{}, fmt.Errorf("invalid byte slice length")
	}
	var addr Addr
	if len(data) == 6 { // IPv4
		addr.IP = net.IPv4(data[0], data[1], data[2], data[3])
		addr.Port = binary.BigEndian.Uint16(data[4:])
	} else {
		addr.IP = net.IP(data[:16])
		addr.Port = binary.BigEndian.Uint16(data[16:])
	}
	return addr, nil
}

func (a *Addr) ToBytesGob() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(a)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *Addr) ToBytesJSON() ([]byte, error) {
	return json.Marshal(a)
}

func (a *Addr) Equals(t *Addr) bool {
	if t == nil || a == nil {
		return t == nil && a == nil
	}
	return a.IP.Equal(t.IP) && a.Port == t.Port
}

type AddrSlice []*Addr

func (a AddrSlice) Equal(b AddrSlice) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !a[i].Equals(b[i]) {
			return false
		}
	}

	return true
}
