package host

import (
	"encoding/binary"
	"fmt"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"net"
	"sync"
)

const (
	MaxRemotes = 10
)

// Cache is an internal struct that splits v4 and v6 addresses inside the cache map
type Cache struct {
	v4 *CacheV4
	v6 *CacheV6
	//relay *CacheRelay
}

func (c *Cache) GetV4() *CacheV4 {
	return c.v4
}

func (c *Cache) GetV46() *CacheV6 {
	return c.v6
}

//func (c *Cache) GetRelay() *CacheRelay {
//	return c.relay
//}

type CacheV4 struct {
	learned  *api.Ipv4Addr
	reported []*api.Ipv4Addr
}

func (c *CacheV4) Learned() *api.Ipv4Addr {
	return c.learned
}

func (c *CacheV4) Reported() []*api.Ipv4Addr {
	return c.reported
}

type CacheV6 struct {
	learned  *api.Ipv6Addr
	reported []*api.Ipv6Addr
}

func (c *CacheV6) Learned() *api.Ipv6Addr {
	return c.learned
}

func (c *CacheV6) Reported() []*api.Ipv6Addr {
	return c.reported
}

type CacheRelay struct {
	relay []uint32
}

func (c *CacheRelay) Relay() []uint32 {
	return c.relay
}

// RemoteList is a unifying concept for lighthouse servers and clients as well as hostinfos.
// It serves as a local cache of query replies, host update notifications, and locally learned addresses
type RemoteList struct {
	// Every interaction with internals requires a lock!
	sync.RWMutex

	// A deduplicated set of addresses. Any accessor should lock beforehand.
	addrs []*udp.Addr

	// These are maps to store v4 and v6 addresses per lighthouse
	// Map key is the vpnIp of the person that told us about this the cached entries underneath.
	// For learned addresses, this is the vpnIp that sent the packet
	cache map[string]*Cache

	// This is a list of remotes that we have tried to handshake with and have returned from the wrong vpn ip.
	// They should not be tried again during a handshake
	badRemotes []*udp.Addr

	// A flag that the cache may have changed and addrs needs to be rebuilt
	shouldRebuild bool
}

func (r *RemoteList) GetCache(name string) *Cache {
	r.RLock()
	defer r.RUnlock()

	if c, ok := r.cache[name]; ok {
		return c
	}
	return nil
}

// CopyAddrs locks and makes a deep copy of the deduplicated address list
func (r *RemoteList) CopyAddrs() []*udp.Addr {
	if r == nil {
		return nil
	}

	r.Rebuild()

	r.RLock()
	defer r.RUnlock()
	c := make([]*udp.Addr, len(r.addrs))
	for i, v := range r.addrs {
		c[i] = v.Copy()
	}
	return c
}

func (r *RemoteList) ForEach(forEach func(addr *udp.Addr)) {
	r.RLock()
	for _, v := range r.addrs {
		forEach(v)
	}
	r.RUnlock()
}

// LearnRemote 锁定并设置拥有者 VPN IP 的已学习地址槽位为提供的 addr。
// 目前仅在调用 HostInfo.SetRemote 时需要使用，因为该方法应涵盖握手和漫游两种情况。
// 它将标记去重后的地址列表为脏状态，因此仅在有新信息可用时调用它。
// TODO: 需要支持允许列表
func (r *RemoteList) LearnRemote(name string, addr *udp.Addr) {
	fmt.Println("LearnRemote")
	r.Lock()
	defer r.Unlock()
	if v4 := addr.IP.To4(); v4 != nil {
		r.unlockedSetLearnedV4(name, api.NewIpv4Addr(v4, uint32(addr.Port)))
	} else {
		r.unlockedSetLearnedV6(name, api.NewIpv6Addr(addr.IP, uint32(addr.Port)))
	}
}

func (r *RemoteList) unlockedSetLearnedV4(name string, to *api.Ipv4Addr) {
	r.shouldRebuild = true
	r.unlockedGetOrMakeV4(name).learned = to
}

// unlockedSetLearnedV4 假定你拥有写锁，并构建缓存和拥有者条目。仅建立 v4 指针。
// 调用者必须在需要时清理学习到的地址缓存。
func (r *RemoteList) unlockedGetOrMakeV4(name string) *CacheV4 {
	am := r.cache[name]
	if am == nil {
		am = &Cache{}
		r.cache[name] = am
	}
	// 如果我们从来没有任何v6地址，请避免占用内存
	if am.v4 == nil {
		am.v4 = &CacheV4{}
	}
	return am.v4
}

func (r *RemoteList) unlockedSetLearnedV6(name string, to *api.Ipv6Addr) {
	r.shouldRebuild = true
	r.unlockedGetOrMakeV6(name).learned = to
}

// unlockedGetOrMakeV4 假定你拥有写锁，并构建缓存和拥有者条目。仅建立 v4 指针。
// 调用者必须在需要时清理学习到的地址缓存。
func (r *RemoteList) unlockedGetOrMakeV6(name string) *CacheV6 {
	am := r.cache[name]
	if am == nil {
		am = &Cache{}
		r.cache[name] = am
	}
	// 如果我们从来没有任何v4地址，请避免占用内存
	if am.v6 == nil {
		am.v6 = &CacheV6{}
	}
	return am.v6
}

// ResetBlockedRemotes 锁定并清除阻止的远程列表
func (r *RemoteList) ResetBlockedRemotes() {
	r.Lock()
	r.badRemotes = nil
	r.Unlock()
}

func (r *RemoteList) UnlockedPrependV4(name string, to *api.Ipv4Addr) {
	r.shouldRebuild = true
	c := r.unlockedGetOrMakeV4(name)

	// 我们正在进行简单的追加，因为这很少被调用
	c.reported = append([]*api.Ipv4Addr{to}, c.reported...)
	if len(c.reported) > MaxRemotes {
		c.reported = c.reported[:MaxRemotes]
	}
}

func (r *RemoteList) UnlockedPrependV6(name string, to *api.Ipv6Addr) {
	r.shouldRebuild = true
	c := r.unlockedGetOrMakeV6(name)

	// 我们正在进行简单的追加，因为这很少被调用
	c.reported = append([]*api.Ipv6Addr{to}, c.reported...)
	if len(c.reported) > MaxRemotes {
		c.reported = c.reported[:MaxRemotes]
	}
}

func (r *RemoteList) Rebuild() {
	r.Lock()
	defer r.Unlock()

	// 仅在缓存更改时重建
	// TODO:shouldRebuild可能毫无意义，因为当灯塔更新到来时，我们不会检查实际的变化
	if r.shouldRebuild {
		r.unlockedCollect()
		r.shouldRebuild = false
	}
}

func (r *RemoteList) unlockedCollect() {
	addrs := r.addrs[:0]
	//relays := r.relays[:0]

	for _, c := range r.cache {
		if c.v4 != nil {
			if c.v4.learned != nil {
				u := NewUDPAddrFromLH4(c.v4.learned)
				if !r.unlockedIsBad(u) {
					addrs = append(addrs, u)
				}
			}

			for _, v := range c.v4.reported {
				u := NewUDPAddrFromLH4(v)
				if !r.unlockedIsBad(u) {
					addrs = append(addrs, u)
				}
			}
		}

		if c.v6 != nil {
			if c.v6.learned != nil {
				u := NewUDPAddrFromLH6(c.v6.learned)
				if !r.unlockedIsBad(u) {
					addrs = append(addrs, u)
				}
			}

			for _, v := range c.v6.reported {
				u := NewUDPAddrFromLH6(v)
				if !r.unlockedIsBad(u) {
					addrs = append(addrs, u)
				}
			}
		}

		//if c.relay != nil {
		//	for _, v := range c.relay.relay {
		//		ip := api.VpnIp(v)
		//		relays = append(relays, &ip)
		//	}
		//}
	}

	r.addrs = addrs
	//r.relays = relays
}

func NewUDPAddrFromLH4(ipp *api.Ipv4Addr) *udp.Addr {
	ip := ipp.Ip
	return udp.NewAddr(
		net.IPv4(byte(ip&0xff000000>>24), byte(ip&0x00ff0000>>16), byte(ip&0x0000ff00>>8), byte(ip&0x000000ff)),
		uint16(ipp.Port),
	)
}

func NewUDPAddrFromLH6(ipp *api.Ipv6Addr) *udp.Addr {
	return udp.NewAddr(lhIp6ToIp(ipp), uint16(ipp.Port))
}

func lhIp6ToIp(v *api.Ipv6Addr) net.IP {
	ip := make(net.IP, 16)
	binary.BigEndian.PutUint64(ip[:8], v.Hi)
	binary.BigEndian.PutUint64(ip[8:], v.Lo)
	return ip
}

// unlockedIsBad 假设您具有写锁定，并检查远程是否与阻止地址列表中的任何条目匹配
func (r *RemoteList) unlockedIsBad(remote *udp.Addr) bool {
	for _, v := range r.badRemotes {
		if v.Equals(remote) {
			return true
		}
	}
	return false
}

func (r *RemoteList) UnlockedSetV4(name string, to []*api.Ipv4Addr) {
	r.shouldRebuild = true
	c := r.unlockedGetOrMakeV4(name)
	// Reset the slice
	c.reported = c.reported[:0]

	// We can't take their array but we can take their pointers
	for _, v := range to[:minInt(len(to), MaxRemotes)] {
		c.reported = append(c.reported, v)
	}
}

func (r *RemoteList) UnlockedSetV6(name string, to []*api.Ipv6Addr) {
	r.shouldRebuild = true
	c := r.unlockedGetOrMakeV6(name)

	// Reset the slice
	c.reported = c.reported[:0]

	// We can't take their array but we can take their pointers
	for _, v := range to[:minInt(len(to), MaxRemotes)] {
		c.reported = append(c.reported, v)
	}
}

//func (r *RemoteList) unlockedGetOrMakeRelay(name string) *CacheRelay {
//	am := r.cache[name]
//	if am == nil {
//		am = &Cache{}
//		r.cache[name] = am
//	}
//	// Avoid occupying memory for relay if we never have any
//	if am.relay == nil {
//		am.relay = &CacheRelay{}
//	}
//	return am.relay
//}

//func (r *RemoteList) UnlockedSetRelay(ownerVpnIp api.VpnIp, vpnIp api.VpnIp, to []uint32) {
//	r.shouldRebuild = true
//	c := r.unlockedGetOrMakeRelay(ownerVpnIp)
//
//	// Reset the slice
//	c.relay = c.relay[:0]
//
//	// We can't take their array but we can take their pointers
//	c.relay = append(c.relay, to[:minInt(len(to), MaxRemotes)]...)
//}

// minInt returns the minimum integer of a or b
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
