package controller

import (
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/host"
	"go.uber.org/zap"
)

const (
	mtu = 9001
)

func (sc *serverController) coalesceAnswers(c *host.Cache, n *api.HostMessage) {
	if v4Cache := c.GetV4(); v4Cache != nil {
		if learned := v4Cache.Learned(); learned != nil {
			n.Ipv4Addr = append(n.Ipv4Addr, learned)
		}
		n.Ipv4Addr = append(n.Ipv4Addr, v4Cache.Reported()...)
	}

	if v6Cache := c.GetV46(); v6Cache != nil {
		if learned := v6Cache.Learned(); learned != nil {
			n.Ipv6Addr = append(n.Ipv6Addr, learned)
		}
		n.Ipv6Addr = append(n.Ipv6Addr, v6Cache.Reported()...)
	}
}

func (sc *serverController) queryAndPrepMessage(name string, f func(cache *host.Cache) (int, error)) (bool, int, error) {
	sc.RLock()
	// Do we have an entry in the main cache?
	if v, ok := sc.addrMap[name]; ok {
		// Swap lh lock for remote list lock
		v.RLock()
		defer v.RUnlock()

		sc.RUnlock()

		// vpnIp should also be the owner here since we are a lighthouse.
		c := v.GetCache(name)
		// Make sure we have
		if c != nil {
			n, err := f(c)
			return true, n, err
		} else {
			sc.logger.Debug("No cache for vpnIp", zap.String("name", name))
		}
		return false, 0, nil
	}
	sc.RUnlock()
	return false, 0, nil
}
