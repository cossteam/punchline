package controller

import (
	"context"
	"fmt"
	apiv1 "github.com/cossteam/punchline/api"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"go.uber.org/zap"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const mtu = 9001

var _ apiv1.Runnable = &PunchController{}

func NewPunchController(
	logger *zap.Logger,
	listenPort uint32,
	hostname string,
	coordinator []*net.UDPAddr,
	amServer bool,
	outside udp.Conn,
	makeupWriter udp.MakeupWriter,
) *PunchController {
	return &PunchController{
		logger:       logger,
		listenPort:   listenPort,
		hostname:     hostname,
		amServer:     amServer,
		coordinator:  coordinator,
		outside:      outside,
		makeupWriter: makeupWriter,
	}
}

type PunchController struct {
	logger *zap.Logger

	listenPort uint32
	hostname   string
	amServer   bool

	makeupWriter udp.MakeupWriter

	outside     udp.Conn
	coordinator []*net.UDPAddr
}

func (pc *PunchController) Start(ctx context.Context) error {
	pc.logger.Info("Starting PunchController")

	//outside, err := udp.NewGenericListener(pc.logger, pc.coordinator[0].IP, int(pc.listenPort))
	//if err != nil {
	//	return err
	//}
	//pc.outside = outside

	go pc.listenOutside()

	clockSource := time.NewTicker(time.Second * time.Duration(10))

	if !pc.amServer {
		//fmt.Println("pc.coordinator[0].IP => ", pc.coordinator[0].IP)
		//makeup, err := udp.DialMakeup(pc.coordinator[0].IP)
		//if err != nil {
		//	return err
		//}
		//
		//pc.makeupWriter = makeup

		go func() {
			defer clockSource.Stop()

			for {
				pc.SendUpdate()

				select {
				case <-ctx.Done():
					return
				case <-clockSource.C:
					continue
				}
			}
		}()
	}

	return nil
}

func (pc *PunchController) HandleRequest(addr net.IPAddr, p []byte) {
	hm := &api.HostMessage{}
	if err := hm.Unmarshal(p); err != nil {
		pc.logger.Error("Failed to unmarshal lighthouse packet",
			zap.Error(err),
		)
		//TODO: send recv_error?
		return
	}

	pc.logger.Info("Received lighthouse packet",
		zap.Any("addr", addr),
		zap.Any("p", p),
		zap.Any("hm", hm),
	)

	switch hm.Type {
	case api.HostMessage_HostQuery:

	case api.HostMessage_HostQueryReply:

	case api.HostMessage_HostUpdateNotification:
		pc.handleHostUpdateNotification(hm, addr)
	case api.HostMessage_HostMovedNotification:

	}
}

func (pc *PunchController) handleHostUpdateNotification(hm *api.HostMessage, addr net.IPAddr) {
	pc.logger.Info("Received host update notification", zap.Any("hm", hm), zap.Any("addr", addr))
}

func (pc *PunchController) SendUpdate() {
	var v4 []*api.Ipv4Addr
	var v6 []*api.Ipv6Addr

	for _, e := range *localIps() {
		//if ip4 := e.To4(); ip4 != nil {
		//	continue
		//}

		fmt.Println("e => ", e)

		// 只添加不是我的VPN/tun IP的IP
		if ip := e.To4(); ip != nil {
			v4 = append(v4, api.NewIpv4Addr(e, pc.listenPort))
		} else {
			v6 = append(v6, api.NewIpv6Addr(e, pc.listenPort))
		}
	}

	m := &api.HostMessage{
		Type:     api.HostMessage_HostUpdateNotification,
		Hostname: pc.hostname,
		Ipv4Addr: v4,
		Ipv6Addr: v6,
	}
	//out := make([]byte, mtu)
	mm, err := m.Marshal()
	if err != nil {
		pc.logger.Error("Error while marshaling for lighthouse update", zap.Error(err))
		return
	}

	for _, v := range pc.coordinator {
		if err := pc.makeupWriter.WriteTo(uint16(pc.listenPort), uint16(v.Port), mm, &udp.Addr{
			IP:   v.IP,
			Port: uint16(v.Port),
		}); err != nil {
			pc.logger.Error("Error while sending lighthouse update", zap.Error(err))
			return
		}
		pc.logger.Debug("正在发送主机更新通知",
			zap.Stringer("lighthouse", v),
			zap.Any("msg", m))
		//lc.interfaceController.EncWriter().SendToVpnIP(header.LightHouse, 0, lighthouse.VpnIp, mm, out)
	}
}

func (pc *PunchController) listenOutside() {
	pc.outside.Listen(func(addr *udp.Addr, out []byte, packet []byte) {
		pc.HandleRequest(net.IPAddr{
			IP: addr.IP,
		}, packet)
	})
}

func localIps() *[]net.IP {
	//FIXME: This function is pretty garbage
	var ips []net.IP
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				//continue
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// TODO: 暂时过滤掉链路本地地址，这是最正确的做法
			// TODO: 也希望能过滤掉基于 SLAAC MAC 地址的 IP
			if ip.IsLoopback() == false && !ip.IsLinkLocalUnicast() {
				ips = append(ips, ip)
			}
		}
	}
	return &ips
}

func (pc *PunchController) Shutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	signal.Notify(sigChan, syscall.SIGINT)

	rawSig := <-sigChan
	sig := rawSig.String()
	pc.logger.Info("Caught signal, shutting down", zap.String("signal", sig))
	pc.outside.Close()
	pc.makeupWriter.Close()
}
