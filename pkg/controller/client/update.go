package controller

import (
	"context"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"github.com/cossteam/punchline/pkg/utils"
	"go.uber.org/zap"
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

	for _, p := range cc.plugins {
		p.Handle(context.Background(), hm)
	}

	if _, err = cc.punchClient.HostUpdate(context.Background(), &api.HostUpdateRequest{
		Hostname: hm.Hostname,
		Ipv4Addr: hm.Ipv4Addr,
		Ipv6Addr: hm.Ipv6Addr,
	}); err != nil {
		cc.logger.Error("Error while sending host update", zap.Error(err))
	}

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
		cc.logger.Debug("正在发送主机更新通知",
			zap.Stringer("lighthouse", v),
			zap.Stringer("ExternalAddr", externalAddr),
			zap.Any("msg", hm))
		//lc.interfaceController.EncWriter().SendToVpnIP(header.LightHouse, 0, lighthouse.VpnIp, mm, out)
	}
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
