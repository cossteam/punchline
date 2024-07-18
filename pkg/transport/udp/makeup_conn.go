package udp

import (
	"encoding/binary"
	"net"
)

const (
	headerLen = 8
)

var _ MakeupWriter = &makeupWriter{}

type makeupWriter struct {
	conn *net.IPConn
}

func DialMakeup(ip net.IP) (MakeupWriter, error) {
	conn, err := net.DialIP("ip4:udp", nil, &net.IPAddr{
		IP: ip,
	})
	if err != nil {
		return nil, err
	}

	return &makeupWriter{
		conn: conn,
	}, nil
}

func (mw *makeupWriter) WriteTo(srcPort uint16, destPort uint16, b []byte, addr *Addr) error {
	packet := append(mw.header(srcPort, destPort, uint16(len(b))), b...)

	_, err := mw.conn.WriteTo(packet, &net.UDPAddr{
		IP:   addr.IP,
		Port: int(addr.Port),
	})
	return err
}

func (mw *makeupWriter) Close() error {
	return mw.conn.Close()
}

func (mw *makeupWriter) header(srcPort, destPort, payloadLen uint16) []byte {
	h := make([]byte, headerLen)
	binary.BigEndian.PutUint16(h[0:], srcPort)
	binary.BigEndian.PutUint16(h[2:], destPort)
	binary.BigEndian.PutUint16(h[4:], headerLen+payloadLen)
	// No checksum.
	binary.BigEndian.PutUint16(h[6:], 0)
	return h
}
