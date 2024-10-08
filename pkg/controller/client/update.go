package controller

import (
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"github.com/cossteam/punchline/pkg/utils"
	"go.uber.org/zap"
	"net"
)

func (cc *clientController) SendUpdate() {
	var v4 []*api.Ipv4Addr
	var v6 []*api.Ipv6Addr

	localIps := utils.LocalIps()
	for _, e := range *localIps {
		if ip := e.To4(); ip != nil {
			v4 = append(v4, api.NewIpv4Addr(e, cc.listenPort))
		} else {
			v6 = append(v6, api.NewIpv6Addr(e, cc.listenPort))
		}
	}

	externalAddrs, err := cc.stunClient.ExternalAddrs()
	if err != nil {
		cc.logger.Error("Error while getting external addresses", zap.Error(err))
	} else {
		for _, a := range externalAddrs {
			if a.IP.To4() != nil {
				v4 = append(v4, api.NewIpv4Addr(a.IP, uint32(a.Port)))
			} else {
				v6 = append(v6, api.NewIpv6Addr(a.IP, uint32(a.Port)))
			}
		}
	}

	hm := &api.HostMessage{
		Type:     api.HostMessage_HostUpdateNotification,
		Hostname: cc.hostname,
		Ipv4Addr: v4,
		Ipv6Addr: v6,
		//ExternalAddr: api.NewIpv4Addr(externalAddr.IP, uint32(externalAddr.Port)),
	}

	externalAddr, err := cc.stunClient.ExternalAddr()
	if err == nil {
		if externalAddr.IP.To4() != nil {
			hm.ExternalAddr = api.NewIpv4Addr(externalAddr.IP, uint32(externalAddr.Port))
		}
	} else {
		cc.logger.Error("Error while getting single external address", zap.Error(err))
	}

	//for _, p := range cc.plugins {
	//	p.Handle(context.Background(), hm)
	//}

	//if _, err = cc.punchClient.HostUpdate(context.Background(), &api.HostUpdateRequest{
	//	Hostname:     hm.Hostname,
	//	Ipv4Addr:     hm.Ipv4Addr,
	//	Ipv6Addr:     hm.Ipv6Addr,
	//	ExternalAddr: hm.ExternalAddr,
	//}); err != nil {
	//	cc.logger.Error("Error while sending host update", zap.Error(err))
	//}

	//out := make([]byte, mtu)
	mm, err := hm.Marshal()
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

		go func(target *net.UDPAddr) {
			var v4addr []*udp.Addr
			for _, vv := range hm.Ipv4Addr {
				v4addr = append(v4addr, utils.NewUDPAddrFromLH4(vv))
			}
			var v6addr []*udp.Addr
			for _, vv := range hm.Ipv6Addr {
				v6addr = append(v6addr, utils.NewUDPAddrFromLH6(vv))
			}

			cc.logger.Debug("发送主机更新通知",
				zap.Stringer("target", target),
				zap.Stringer("externalAddr", externalAddr),
				zap.Any("v4Addr", v4addr),
				zap.Any("v6Addr", v4addr),
			)
		}(v)

		//lc.interfaceController.EncWriter().SendToVpnIP(header.LightHouse, 0, lighthouse.VpnIp, mm, out)
	}
}

func (cc *clientController) createHostMessage() (*api.HostMessage, error) {
	var v4 []*api.Ipv4Addr
	var v6 []*api.Ipv6Addr

	// 获取本地IP地址
	localIps := utils.LocalIps()
	for _, e := range *localIps {
		if ip := e.To4(); ip != nil {
			v4 = append(v4, api.NewIpv4Addr(e, cc.listenPort))
		} else {
			v6 = append(v6, api.NewIpv6Addr(e, cc.listenPort))
		}
	}

	// 获取外部地址
	externalAddrs, err := cc.stunClient.ExternalAddrs()
	if err != nil {
		cc.logger.Error("Error while getting external addresses", zap.Error(err))
		return nil, err
	} else {
		for _, a := range externalAddrs {
			if a.IP.To4() != nil {
				v4 = append(v4, api.NewIpv4Addr(a.IP, uint32(a.Port)))
			} else {
				v6 = append(v6, api.NewIpv6Addr(a.IP, uint32(a.Port)))
			}
		}
	}

	// 创建 HostMessage
	hm := &api.HostMessage{
		Type:     api.HostMessage_HostUpdateNotification,
		Hostname: cc.hostname,
		Ipv4Addr: v4,
		Ipv6Addr: v6,
	}

	// 获取单个外部地址
	externalAddr, err := cc.stunClient.ExternalAddr()
	if err != nil {
		cc.logger.Error("Error while getting single external address", zap.Error(err))
		return nil, err
	}

	if externalAddr.IP.To4() != nil {
		hm.ExternalAddr = api.NewIpv4Addr(externalAddr.IP, uint32(externalAddr.Port))
	}

	return hm, nil
}

//func (cc *clientController) SendUpdate() {
//	var v4 []*api.Ipv4Addr
//	var v6 []*api.Ipv6Addr
//
//	for _, e := range *utils.LocalIps() {
//		//if ip4 := e.To4(); ip4 != nil {
//		//	continue
//		//}
//
//		//fmt.Println("e => ", e)
//
//		// 只添加不是我的VPN/tun IP的IP
//		if ip := e.To4(); ip != nil {
//			v4 = append(v4, api.NewIpv4Addr(e, cc.listenPort))
//		} else {
//			v6 = append(v6, api.NewIpv6Addr(e, cc.listenPort))
//		}
//	}
//
//	addrs, err := cc.stunClient.ExternalAddrs()
//	if err != nil {
//		cc.logger.Error("Error while getting external addresses", zap.Error(err))
//	} else {
//		for _, a := range addrs {
//			v4 = append(v4, api.NewIpv4Addr(a.IP, uint32(a.Port)))
//		}
//	}
//
//	m := &api.HostMessage{
//		Type:     api.HostMessage_HostUpdateNotification,
//		Hostname: cc.hostname,
//		Ipv4Addr: v4,
//		Ipv6Addr: v6,
//	}
//
//	addr, err := cc.stunClient.ExternalAddr()
//	if err == nil {
//		m.ExternalAddr = api.NewIpv4Addr(addr.IP, uint32(addr.Port))
//	}
//
//	//out := make([]byte, mtu)
//	mm, err := m.Marshal()
//	if err != nil {
//		cc.logger.Error("Error while marshaling for lighthouse update", zap.Error(err))
//		return
//	}
//
//	for _, v := range cc.coordinator {
//		if err := cc.makeupWriter.WriteTo(uint16(cc.listenPort), uint16(v.Port), mm, &udp.Addr{
//			IP:   v.IP,
//			Port: uint16(v.Port),
//		}); err != nil {
//			cc.logger.Error("Error while sending lighthouse update", zap.Error(err))
//			return
//		}
//		cc.logger.Debug("正在发送主机更新通知",
//			zap.Stringer("lighthouse", v),
//			zap.Any("msg", m))
//		//lc.interfaceController.EncWriter().SendToVpnIP(header.LightHouse, 0, lighthouse.VpnIp, mm, out)
//	}
//}
