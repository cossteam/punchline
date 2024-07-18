package udp

import (
	"fmt"
	"go.uber.org/zap"
	"net"
)

type GenericConn struct {
	*net.UDPConn
	l *zap.Logger
}

func (u *GenericConn) LocalAddr() (*Addr, error) {
	a := u.UDPConn.LocalAddr()

	switch v := a.(type) {
	case *net.UDPAddr:
		addr := &Addr{IP: make([]byte, len(v.IP))}
		copy(addr.IP, v.IP)
		addr.Port = uint16(v.Port)
		return addr, nil

	default:
		return nil, fmt.Errorf("LocalAddr returned: %#v", a)
	}
}

func (u *GenericConn) Listen(r EncReader) {
	plaintext := make([]byte, MTU)
	buffer := make([]byte, MTU)
	udpAddr := &Addr{IP: make([]byte, 16)}

	for {
		// Just read one packet at a time
		n, rua, err := u.ReadFromUDP(buffer)
		if err != nil {
			u.l.Debug("udp socket is closed, exiting read loop", zap.Error(err))
			return
		}

		udpAddr.IP = rua.IP
		udpAddr.Port = uint16(rua.Port)
		r(udpAddr, plaintext[:n], buffer[:n])
	}
}

func (u *GenericConn) WriteTo(b []byte, addr *Addr) error {
	_, err := u.UDPConn.WriteToUDP(b, &net.UDPAddr{IP: addr.IP, Port: int(addr.Port)})
	return err
}

func NewGenericListener(logger *zap.Logger, ip net.IP, port int) (Conn, error) {
	addr := &net.UDPAddr{IP: ip, Port: port}
	udpConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	genericConn := &GenericConn{
		UDPConn: udpConn,
		l:       logger,
	}

	return genericConn, nil
}
