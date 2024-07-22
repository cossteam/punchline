package udp

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	headerLen = 8
)

var _ MakeupWriter = &makeupWriter{}

type makeupWriter struct {
	conn *net.IPConn
}

func DialMakeup(ip net.IP, port int) (MakeupWriter, error) {
	//raddr := &net.UDPAddr{
	//	IP:   ip,
	//	Port: port,
	//}
	//conn, err := net.DialUDP("udp4", nil, raddr)
	//if err != nil {
	//	return nil, err
	//}
	conn, err := net.DialIP("ip4:udp", nil, &net.IPAddr{
		IP: ip,
	})
	if err != nil {
		return nil, err
	}

	//fmt.Println("ip => ", ip)

	return &makeupWriter{
		conn: conn,
	}, nil
}

func (mw *makeupWriter) WriteTo(srcPort uint16, destPort uint16, b []byte, addr *Addr) error {

	packet := append(mw.header(srcPort, destPort, uint16(len(b))), b...)

	//fmt.Println("b len => ", len(b))
	//fmt.Println("packet => ", packet)
	//fmt.Println("packet len => ", len(packet))
	//printUDPHeader(packet)

	_, err := mw.conn.Write(packet)
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

func printUDPHeader(header []byte) {
	if len(header) < 8 {
		fmt.Println("Invalid UDP header")
		return
	}
	srcPort := binary.BigEndian.Uint16(header[0:2])
	destPort := binary.BigEndian.Uint16(header[2:4])
	length := binary.BigEndian.Uint16(header[4:6])
	checksum := binary.BigEndian.Uint16(header[6:8])

	fmt.Printf("UDP Header:\n")
	fmt.Printf("Source Port: %d\n", srcPort)
	fmt.Printf("Destination Port: %d\n", destPort)
	fmt.Printf("Length: %d\n", length)
	fmt.Printf("Checksum: 0x%x\n", checksum)
}
