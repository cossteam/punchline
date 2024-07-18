package controller

import (
	"context"
	"fmt"
	apiv1 "github.com/cossteam/punchline/api"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"go.uber.org/zap"
	"net"
	"time"
)

var _ apiv1.Runnable = &client{}

func NewClient(
	logger *zap.Logger,
	listenPort uint32,
	hostname string,
	makeupWriter udp.MakeupWriter,
	coordinator []*net.UDPAddr,
) apiv1.Runnable {
	return &client{
		logger:       logger,
		listenPort:   listenPort,
		hostname:     hostname,
		makeupWriter: makeupWriter,
		coordinator:  coordinator,
	}
}

type client struct {
	logger       *zap.Logger
	listenPort   uint32
	hostname     string
	makeupWriter udp.MakeupWriter
	coordinator  []*net.UDPAddr
}

func (c *client) Start(ctx context.Context) error {
	c.logger.Info("Starting client")
	clockSource := time.NewTicker(time.Second * time.Duration(10))

	//go func() {
	defer clockSource.Stop()

	for {
		c.SendUpdate()

		select {
		case <-ctx.Done():
			return nil
		case <-clockSource.C:
			continue
		}
	}
	//}()

	//return nil
}

func (c *client) SendUpdate() {
	var v4 []*api.Ipv4Addr
	var v6 []*api.Ipv6Addr

	for _, e := range *localIps() {
		//if ip4 := e.To4(); ip4 != nil {
		//	continue
		//}

		fmt.Println("e => ", e)

		// 只添加不是我的VPN/tun IP的IP
		if ip := e.To4(); ip != nil {
			v4 = append(v4, api.NewIpv4Addr(e, c.listenPort))
		} else {
			v6 = append(v6, api.NewIpv6Addr(e, c.listenPort))
		}
	}

	m := &api.HostMessage{
		Type:     api.HostMessage_HostUpdateNotification,
		Hostname: c.hostname,
		Ipv4Addr: v4,
		Ipv6Addr: v6,
	}
	//out := make([]byte, mtu)
	mm, err := m.Marshal()
	if err != nil {
		c.logger.Error("Error while marshaling for lighthouse update", zap.Error(err))
		return
	}

	for _, v := range c.coordinator {
		if err := c.makeupWriter.WriteTo(uint16(c.listenPort), uint16(v.Port), mm, &udp.Addr{
			IP:   v.IP,
			Port: uint16(v.Port),
		}); err != nil {
			c.logger.Error("Error while sending lighthouse update", zap.Error(err))
			return
		}
		c.logger.Debug("正在发送主机更新通知",
			zap.Stringer("lighthouse", v),
			zap.Any("msg", m))
		//lc.interfaceController.EncWriter().SendToVpnIP(header.LightHouse, 0, lighthouse.VpnIp, mm, out)
	}
}
