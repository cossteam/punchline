package controller

import (
	"context"
	"fmt"
	apiv1 "github.com/cossteam/punchline/api"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/config"
	"github.com/cossteam/punchline/pkg/host"
	plugin "github.com/cossteam/punchline/pkg/plugin/client"
	"github.com/cossteam/punchline/pkg/publisher"
	stunclient "github.com/cossteam/punchline/pkg/sutn"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"github.com/cossteam/punchline/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"time"
)

var _ apiv1.Runnable = &clientController{}

func NewClientController(
	logger *zap.Logger,
	listenPort uint32,
	hostname string,
	makeupWriter udp.MakeupWriter,
	coordinator []*net.UDPAddr,
	c *config.Config,
	opts ...ClientOption,
) apiv1.Runnable {
	cc := &clientController{
		logger:       logger,
		listenPort:   listenPort,
		hostname:     hostname,
		makeupWriter: makeupWriter,
		coordinator:  coordinator,
		c:            c,
	}
	for _, opt := range opts {
		opt(cc)
	}
	return cc
}

type ClientOption func(*clientController)

type clientController struct {
	c            *config.Config
	logger       *zap.Logger
	listenPort   uint32
	hostname     string
	makeupWriter udp.MakeupWriter

	//punchConn udp.MakeupWriter
	coordinator []*net.UDPAddr

	hostMap *host.HostMap

	plugins     []plugin.Plugin
	stunClient  stunclient.STUNClient
	pubClient   publisher.PublisherClient
	punchClient api.PunchServiceClient
}

func (cc *clientController) Start(ctx context.Context) error {
	cc.logger.Info("Starting clientController")

	serverShutdown := make(chan struct{})
	go func() {
		<-ctx.Done()
		cc.logger.Info("Shutting down Client")
		if err := cc.pubClient.Close(); err != nil {
			cc.logger.Error("Failed to close Client", zap.Error(err))
		}
		close(serverShutdown)
	}()

	stunClient, err := stunclient.NewClient(cc.c.StunServer)
	if err != nil {
		cc.logger.Error("Failed to create STUN client", zap.Error(err))
		return err
	}
	defer stunClient.Close()
	cc.stunClient = stunClient

	conn, err := grpc.NewClient(cc.c.Publisher.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	cc.punchClient = api.NewPunchServiceClient(conn)

	if err := cc.InitAndSubscribe(); err != nil {
		cc.logger.Error("Failed to init and subscribe", zap.Error(err))
		return err
	}

	go func() {
		clockSource := time.NewTicker(time.Second * time.Duration(30))
		defer clockSource.Stop()

		for {
			cc.SendUpdate()

			select {
			case <-ctx.Done():
				return
			case <-clockSource.C:
				continue
			}
		}
	}()

	<-serverShutdown

	return nil
}

// InitAndSubscribe 初始化并订阅主题
func (cc *clientController) InitAndSubscribe() error {
	cc.logger.Info("Initializing publisher", zap.Any("subscriptions", cc.c.Subscriptions))
	pubSubServiceClient, err := publisher.NewClient(cc.c.Publisher.Addr, publisher.WithClientName(cc.c.Hostname))
	if err != nil {
		return fmt.Errorf("failed to create publisher clientController: %w", err)
	}
	cc.pubClient = pubSubServiceClient

	for _, sub := range cc.c.Subscriptions {
		if err := cc.pubClient.Subscribe(context.Background(), sub.Topic, func(message *publisher.Message) error {
			return cc.handleSubscribe(message)
		}); err != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", sub.Topic, err)
		}
	}

	return nil
}

func (cc *clientController) handleSubscribe(message *publisher.Message) error {
	hm := &api.HostMessage{}
	if err := hm.Unmarshal(message.Data); err != nil {
		cc.logger.Error("Error while unmarshalling message data", zap.Error(err))
		return err
	}

	cc.logger.Info("收到订阅消息",
		zap.Any("message", message),
		zap.Any("hm", hm),
	)

	go func() {
		for _, p := range cc.plugins {
			p.Handle(context.Background(), hm)
		}
	}()

	switch hm.Type {
	case api.HostMessage_HostOnlineNotification:
		cc.handleHostOnlineNotification(hm)
	case api.HostMessage_HostPunchNotification:
		cc.handleHostPunchNotification(hm)
	}

	return nil
}

func (cc *clientController) handleHostOnlineNotification(hm *api.HostMessage) {
	cc.logger.Debug("收到主机上线通知", zap.Any("hm", hm), zap.Any("makeupPort", cc.listenPort))
	cc.handleHostPunchNotification(hm)
}

func (cc *clientController) handleHostPunchNotification(hm *api.HostMessage) {
	cc.logger.Debug("收到主机打洞通知", zap.Any("hm", hm), zap.Any("makeupPort", cc.listenPort))
	//empty := []byte{0}
	newHm := &api.HostMessage{
		Type:     api.HostMessage_None,
		Hostname: "empty",
	}
	empty, err := newHm.Marshal()
	if err != nil {
		cc.logger.Error("Error while marshalling for lighthouse update", zap.Error(err))
		return
	}

	punch := func(vpnPeer *udp.Addr) {
		time.Sleep(time.Second)
		if err := cc.makeupWriter.WriteTo(uint16(cc.listenPort), vpnPeer.Port, empty, vpnPeer); err != nil {
			cc.logger.Error("Error while sending lighthouse punch", zap.Error(err))
		} else {
			cc.logger.Debug(fmt.Sprintf("Punching on %s for %s", vpnPeer.String(), hm.Hostname))
		}
	}

	for _, a := range hm.Ipv4Addr {
		vpnPeer := utils.NewUDPAddrFromLH4(a)
		if vpnPeer != nil {
			go punch(vpnPeer)
		}
	}

	for _, a := range hm.Ipv6Addr {
		vpnPeer := utils.NewUDPAddrFromLH6(a)
		if vpnPeer != nil {
			go punch(vpnPeer)
		}
	}
}
