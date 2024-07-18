package udp

const MTU = 9001

type EncReader func(addr *Addr, out []byte, packet []byte)

type Conn interface {
	LocalAddr() (*Addr, error)
	Listen(r EncReader)
	WriteTo(b []byte, addr *Addr) error
	Close() error
}

type MakeupWriter interface {
	WriteTo(srcPort uint16, destPort uint16, b []byte, addr *Addr) error
	Close() error
}
