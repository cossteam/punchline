package controller

import (
	"context"
	"fmt"
	apiv1 "github.com/cossteam/punchline/api"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/config"
	"github.com/cossteam/punchline/pkg/host"
	"github.com/cossteam/punchline/pkg/publisher"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"go.uber.org/zap"
	"net"
	"time"
)

var _ apiv1.Runnable = &clientController{}

func NewClient(
	logger *zap.Logger,
	listenPort uint32,
	hostname string,
	makeupWriter udp.MakeupWriter,
	coordinator []*net.UDPAddr,
	c *config.Config,
) apiv1.Runnable {
	return &clientController{
		logger:       logger,
		listenPort:   listenPort,
		hostname:     hostname,
		makeupWriter: makeupWriter,
		coordinator:  coordinator,
		c:            c,
	}
}

type clientController struct {
	c            *config.Config
	logger       *zap.Logger
	listenPort   uint32
	hostname     string
	makeupWriter udp.MakeupWriter

	//punchConn udp.MakeupWriter
	coordinator []*net.UDPAddr

	hostMap *host.HostMap

	pubClient publisher.PublisherClient
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

	if err := cc.InitAndSubscribe(); err != nil {
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

	switch hm.Type {
	case api.HostMessage_HostPunchNotification:
		cc.handleHostPunchNotification(hm)
	}

	return nil
}

func (cc *clientController) handleHostPunchNotification(hm *api.HostMessage) {
	cc.logger.Debug("收到主机打洞通知", zap.Any("hm", hm))
	empty := []byte{0}

	punch := func(vpnPeer *udp.Addr) {
		time.Sleep(time.Second)
		if err := cc.makeupWriter.WriteTo(uint16(cc.listenPort), vpnPeer.Port, empty, vpnPeer); err != nil {
			cc.logger.Error("Error while sending lighthouse punch", zap.Error(err))
		} else {
			cc.logger.Debug(fmt.Sprintf("Punching on %s for %s", vpnPeer.String(), hm.Hostname))
		}
	}

	for _, a := range hm.Ipv4Addr {
		vpnPeer := NewUDPAddrFromLH4(a)
		if vpnPeer != nil {
			go punch(vpnPeer)
		}
	}

	for _, a := range hm.Ipv6Addr {
		vpnPeer := NewUDPAddrFromLH6(a)
		if vpnPeer != nil {
			go punch(vpnPeer)
		}
	}
}

func (cc *clientController) SendUpdate() {
	var v4 []*api.Ipv4Addr
	var v6 []*api.Ipv6Addr

	for _, e := range *localIps() {
		//if ip4 := e.To4(); ip4 != nil {
		//	continue
		//}

		//fmt.Println("e => ", e)

		// 只添加不是我的VPN/tun IP的IP
		if ip := e.To4(); ip != nil {
			v4 = append(v4, api.NewIpv4Addr(e, cc.listenPort))
		} else {
			v6 = append(v6, api.NewIpv6Addr(e, cc.listenPort))
		}
	}

	m := &api.HostMessage{
		Type:     api.HostMessage_HostUpdateNotification,
		Hostname: cc.hostname,
		Ipv4Addr: v4,
		Ipv6Addr: v6,
	}
	//out := make([]byte, mtu)
	mm, err := m.Marshal()
	if err != nil {
		cc.logger.Error("Error while marshaling for lighthouse update", zap.Error(err))
		return
	}

	for _, v := range cc.coordinator {
		if err := cc.makeupWriter.WriteTo(uint16(cc.listenPort), uint16(v.Port), mm, &udp.Addr{
			IP:   v.IP,
			Port: uint16(v.Port),
		}); err != nil {
			cc.logger.Error("Error while sending lighthouse update", zap.Error(err))
			return
		}
		cc.logger.Debug("正在发送主机更新通知",
			zap.Stringer("lighthouse", v),
			zap.Any("msg", m))
		//lc.interfaceController.EncWriter().SendToVpnIP(header.LightHouse, 0, lighthouse.VpnIp, mm, out)
	}
}
