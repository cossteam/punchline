package controller

import (
	"context"
	"fmt"
	apiv1 "github.com/cossteam/punchline/api"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"go.uber.org/zap"
	"net"
)

var _ apiv1.Runnable = &server{}

func NewServer(
	logger *zap.Logger,
	listenPort uint32,
	hostname string,
	outside udp.Conn,
) apiv1.Runnable {
	return &server{
		logger:     logger,
		listenPort: listenPort,
		hostname:   hostname,
		outside:    outside,
	}
}

type server struct {
	logger     *zap.Logger
	listenPort uint32
	hostname   string
	outside    udp.Conn
}

func (s *server) Start(ctx context.Context) error {
	serverShutdown := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down Server")

		if err := s.outside.Close(); err != nil {
			s.logger.Error("Failed to close Server", zap.Error(err))
		}
		close(serverShutdown)
	}()

	addr, err := s.outside.LocalAddr()
	if err != nil {
		return err
	}
	s.logger.Info("Starting Server", zap.Any("addr", addr))
	go s.listenOutside()

	<-serverShutdown

	fmt.Println("sfnbksngb")

	return nil
}

func (s *server) listenOutside() {
	s.outside.Listen(func(addr *udp.Addr, out []byte, packet []byte) {
		s.HandleRequest(net.IPAddr{
			IP: addr.IP,
		}, packet)
	})
}

func (s *server) HandleRequest(addr net.IPAddr, p []byte) {
	hm := &api.HostMessage{}
	if err := hm.Unmarshal(p); err != nil {
		s.logger.Error("Failed to unmarshal lighthouse packet",
			zap.Error(err),
		)
		//TODO: send recv_error?
		return
	}

	s.logger.Info("Received lighthouse packet",
		zap.Any("addr", addr),
		zap.Any("p", p),
		zap.Any("hm", hm),
	)

	switch hm.Type {
	case api.HostMessage_HostQuery:

	case api.HostMessage_HostQueryReply:

	case api.HostMessage_HostUpdateNotification:
		s.handleHostUpdateNotification(hm, addr)
	case api.HostMessage_HostMovedNotification:

	}
}

func (s *server) handleHostUpdateNotification(hm *api.HostMessage, addr net.IPAddr) {
	s.logger.Info("Received host update notification", zap.Any("hm", hm), zap.Any("addr", addr))
}
