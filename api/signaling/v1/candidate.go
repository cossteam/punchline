package signaling

import (
	"fmt"
	"github.com/pion/ice/v2"
)

func NewConnectionState(cs ice.ConnectionState) ConnectionState {
	switch cs {
	case ice.ConnectionStateNew:
		return ConnectionState_NEW
	case ice.ConnectionStateChecking:
		return ConnectionState_CHECKING
	case ice.ConnectionStateConnected:
		return ConnectionState_CONNECTED
	case ice.ConnectionStateCompleted:
		return ConnectionState_COMPLETED
	case ice.ConnectionStateFailed:
		return ConnectionState_FAILED
	case ice.ConnectionStateDisconnected:
		return ConnectionState_DISCONNECTED
	case ice.ConnectionStateClosed:
		return ConnectionState_CLOSED
	default:
		panic("unknown connection state")
	}
}

func NewCandidate(ic ice.Candidate) *Candidate {
	c := &Candidate{
		Type:        CandidateType(ic.Type()),
		Foundation:  ic.Foundation(),
		Component:   int32(ic.Component()),
		NetworkType: NetworkType(ic.NetworkType()),
		Priority:    int32(ic.Priority()),
		Address:     ic.Address(),
		Port:        int32(ic.Port()),
		TcpType:     TCPType(ic.TCPType()),
	}

	if r := ic.RelatedAddress(); r != nil {
		c.RelatedAddress = &RelatedAddress{
			Address: r.Address,
			Port:    int32(r.Port),
		}
	}

	if rc, ok := ic.(*ice.CandidateRelay); ok {
		c.RelayProtocol = NewProtocol(rc.RelayProtocol())
	}

	return c
}

func NewProtocol(rp string) RelayProtocol {
	switch rp {
	case "udp", "UDP":
		return RelayProtocol_UDP
	case "tcp":
		return RelayProtocol_TCP
	case "dtls":
		return RelayProtocol_DTLS
	case "tls":
		return RelayProtocol_TLS
	}

	return -1
}

func (p RelayProtocol) ToString() string {
	switch p {
	case RelayProtocol_UDP:
		return "udp"
	case RelayProtocol_TCP:
		return "tcp"
	case RelayProtocol_DTLS:
		return "dtls"
	case RelayProtocol_TLS:
		return "tls"
	case RelayProtocol_UNSPECIFIED_RELAY_PROTOCOL:
	}

	return "unknown"
}

func (c *Candidate) ICECandidate() (ice.Candidate, error) {
	var err error

	relAddr := ""
	relPort := 0
	if c.RelatedAddress != nil {
		relAddr = c.RelatedAddress.Address
		relPort = int(c.RelatedAddress.Port)
	}

	nw := ice.NetworkType(c.NetworkType)

	var ic ice.Candidate
	switch c.Type {
	case CandidateType_HOST:
		ic, err = ice.NewCandidateHost(&ice.CandidateHostConfig{
			Network:    nw.String(),
			Address:    c.Address,
			Port:       int(c.Port),
			Component:  uint16(c.Component),
			Priority:   uint32(c.Priority),
			Foundation: c.Foundation,
			TCPType:    ice.TCPType(c.TcpType),
		})
	case CandidateType_SERVER_REFLEXIVE:
		ic, err = ice.NewCandidateServerReflexive(&ice.CandidateServerReflexiveConfig{
			Network:    nw.String(),
			Address:    c.Address,
			Port:       int(c.Port),
			Component:  uint16(c.Component),
			Priority:   uint32(c.Priority),
			Foundation: c.Foundation,
			RelAddr:    relAddr,
			RelPort:    relPort,
		})
	case CandidateType_PEER_REFLEXIVE:
		ic, err = ice.NewCandidatePeerReflexive(&ice.CandidatePeerReflexiveConfig{
			Network:    nw.String(),
			Address:    c.Address,
			Port:       int(c.Port),
			Component:  uint16(c.Component),
			Priority:   uint32(c.Priority),
			Foundation: c.Foundation,
			RelAddr:    relAddr,
			RelPort:    relPort,
		})

	case CandidateType_RELAY:
		ic, err = ice.NewCandidateRelay(&ice.CandidateRelayConfig{
			Network:       nw.String(),
			Address:       c.Address,
			Port:          int(c.Port),
			Component:     uint16(c.Component),
			Priority:      uint32(c.Priority),
			Foundation:    c.Foundation,
			RelAddr:       relAddr,
			RelPort:       relPort,
			RelayProtocol: c.RelayProtocol.ToString(),
		})

	default:
		err = fmt.Errorf("%w: %s", ice.ErrUnknownCandidateTyp, c.Type)
	}

	return ic, err
}
