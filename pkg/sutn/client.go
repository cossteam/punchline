package stunclient

import (
	"fmt"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"github.com/pion/stun"
	"net"
	"sync"
)

// STUNClient 是一个用于与 STUN 服务器通信的接口。
type STUNClient interface {
	XORMappedAddress() (*udp.Addr, error)

	MappedAddress() (*udp.Addr, error)

	ChangedAddress() (*udp.Addr, error)

	OtherAddress() (*udp.Addr, error)

	ExternalAddr() (*udp.Addr, error)

	ExternalAddrs() ([]*udp.Addr, error)

	Close() error
}

var _ STUNClient = &stunClient{}

// ChangedAddress 是用于解析 CHANGED-ADDRESS 属性的自定义类型。
type ChangedAddress struct {
	IP   net.IP
	Port int
}

func (s *ChangedAddress) GetFrom(message *stun.Message) error {
	a := (*stun.MappedAddress)(s)
	return a.GetFromAs(message, stun.AttrChangedAddress)
}

func NewClient(stunURI string) (STUNClient, error) {
	u, err := stun.ParseURI(stunURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse STUN URI: %w", err)
	}

	conn, err := stun.DialURI(u, &stun.DialConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to dial STUN server: %w", err)
	}

	return &stunClient{conn: conn}, nil
}

type stunClient struct {
	conn *stun.Client
}

func (c *stunClient) ExternalAddr() (*udp.Addr, error) {
	// Build the STUN Binding Request message
	message, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to build STUN request: %w", err)
	}

	var xorAddr stun.XORMappedAddress
	var mappedAddr stun.MappedAddress
	var resultAddr *udp.Addr
	var resultErr error

	// Function to process STUN response
	processResponse := func(res stun.Event) {
		if res.Error != nil {
			resultErr = res.Error
			return
		}
		if err := xorAddr.GetFrom(res.Message); err == nil {
			// Prefer XORMappedAddress if available
			resultAddr = &udp.Addr{
				IP:   xorAddr.IP,
				Port: uint16(xorAddr.Port),
			}
		} else if err := mappedAddr.GetFrom(res.Message); err == nil {
			// Fallback to MappedAddress if XORMappedAddress not available
			if resultAddr == nil {
				resultAddr = &udp.Addr{
					IP:   mappedAddr.IP,
					Port: uint16(mappedAddr.Port),
				}
			}
		}
	}

	// Send the request and process the response
	if err = c.conn.Do(message, processResponse); err != nil {
		return nil, fmt.Errorf("failed to get external address: %w", err)
	}

	if resultErr != nil {
		return nil, resultErr
	}

	return resultAddr, nil
}

func (c *stunClient) ExternalAddrs() ([]*udp.Addr, error) {
	var wg sync.WaitGroup
	addrMap := make(map[string]*udp.Addr)
	var resultErr error

	// Fetch addresses using ExternalAddr
	wg.Add(1)
	go func() {
		defer wg.Done()
		addr, err := c.ExternalAddr()
		if err != nil {
			resultErr = fmt.Errorf("failed to get external address: %w", err)
			return
		}
		if addr != nil && addr.IP != nil {
			key := fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)
			addrMap[key] = addr
		}
	}()

	// Fetch addresses using OtherAddress
	//wg.Add(1)
	//go func() {
	//	defer wg.Done()
	//	otherAddr, err := c.OtherAddress()
	//	if err != nil {
	//		resultErr = fmt.Errorf("failed to get other address: %w", err)
	//		return
	//	}
	//
	//	if otherAddr == nil || otherAddr.IP == nil {
	//		return
	//	}
	//
	//	// Use the address from OtherAddress as a new STUN server
	//	newStunURI := fmt.Sprintf("stun:%s:%d", otherAddr.IP.String(), otherAddr.Port)
	//	newClient, err := NewClient(newStunURI)
	//	if err != nil {
	//		resultErr = fmt.Errorf("failed to create new STUN client: %w", err)
	//		return
	//	}
	//	defer newClient.Close()
	//
	//	// Fetch addresses using the new STUN client
	//	newAddr, err := newClient.ExternalAddr()
	//	if err != nil {
	//		resultErr = fmt.Errorf("failed to get external address from new STUN client: %w", err)
	//		return
	//	}
	//	if newAddr != nil && newAddr.IP != nil {
	//		key := fmt.Sprintf("%s:%d", newAddr.IP.String(), newAddr.Port)
	//		addrMap[key] = newAddr
	//	}
	//}()

	// Wait for all goroutines to finish
	wg.Wait()

	if resultErr != nil {
		return nil, resultErr
	}

	// Convert the map to a slice
	uniqueAddrs := make([]*udp.Addr, 0, len(addrMap))
	for _, addr := range addrMap {
		uniqueAddrs = append(uniqueAddrs, addr)
	}

	return uniqueAddrs, nil
}

func (c *stunClient) XORMappedAddress() (*udp.Addr, error) {
	var xorAddr stun.XORMappedAddress
	err := c.sendRequest(&xorAddr)
	if err != nil {
		return nil, err
	}
	return &udp.Addr{
		IP:   xorAddr.IP,
		Port: uint16(xorAddr.Port),
	}, nil
}

func (c *stunClient) MappedAddress() (*udp.Addr, error) {
	var mappedAddr stun.MappedAddress
	err := c.sendRequest(&mappedAddr)
	if err != nil {
		return nil, err
	}
	return &udp.Addr{
		IP:   mappedAddr.IP,
		Port: uint16(mappedAddr.Port),
	}, nil
}

func (c *stunClient) ChangedAddress() (*udp.Addr, error) {
	var changedAddr ChangedAddress
	err := c.sendRequest(&changedAddr)
	if err != nil {
		return nil, err
	}
	return &udp.Addr{
		IP:   changedAddr.IP,
		Port: uint16(changedAddr.Port),
	}, nil
}

func (c *stunClient) OtherAddress() (*udp.Addr, error) {
	var otherAddr stun.OtherAddress
	var changedAddr ChangedAddress
	var resultAddr *udp.Addr
	var resultErr error

	// Build the STUN Binding Request message
	message, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to build STUN request: %w", err)
	}

	// Function to process STUN response
	processResponse := func(res stun.Event) {
		if res.Error != nil {
			resultErr = res.Error
			return
		}

		if err := otherAddr.GetFrom(res.Message); err == nil {
			resultAddr = &udp.Addr{
				IP:   otherAddr.IP,
				Port: uint16(otherAddr.Port),
			}
			return
		}

		if err := changedAddr.GetFrom(res.Message); err == nil && resultAddr == nil {
			resultAddr = &udp.Addr{
				IP:   changedAddr.IP,
				Port: uint16(changedAddr.Port),
			}
		}
	}

	// Send the request and process the response
	if err = c.conn.Do(message, processResponse); err != nil {
		return nil, fmt.Errorf("failed to get OtherAddress: %w", err)
	}

	if resultErr != nil {
		return nil, resultErr
	}

	return resultAddr, nil
}

func (c *stunClient) sendRequest(addr interface{}) error {
	message, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return fmt.Errorf("failed to build STUN request: %w", err)
	}

	err = c.conn.Do(message, func(res stun.Event) {
		if res.Error != nil {
			err = res.Error
			return
		}
		switch v := addr.(type) {
		case *stun.XORMappedAddress:
			err = v.GetFrom(res.Message)
		case *stun.MappedAddress:
			err = v.GetFrom(res.Message)
		case *ChangedAddress:
			err = v.GetFrom(res.Message)
		case *stun.OtherAddress:
			err = v.GetFrom(res.Message)
		default:
			err = fmt.Errorf("unsupported address type")
		}
	})
	return err
}

func (c *stunClient) Close() error {
	return c.conn.Close()
}
