package host

import (
	"encoding/json"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"go.uber.org/zap"
	"net"
	"sync"
)

func NewHostMap(logger *zap.Logger) *HostMap {
	h := map[string]*HostInfo{}
	m := HostMap{
		Hosts:  h,
		logger: logger,
	}

	return &m
}

type HostMap struct {
	sync.RWMutex //Because we concurrently read and write to our maps
	//Indexes         map[uint32]*HostInfo
	//Relays          map[uint32]*HostInfo // Maps a Relay IDX to a Relay HostInfo object
	//RemoteIndexes   map[uint32]*HostInfo
	Hosts           map[string]*HostInfo
	logger          *zap.Logger
	preferredRanges []*net.IPNet
	metricsEnabled  bool
}

func (hm *HostMap) GetHost(name string) *HostInfo {
	hm.RLock()
	h, ok := hm.Hosts[name]
	hm.RUnlock()
	if ok {
		return h
	}
	return nil
}

func (hm *HostMap) AddHost(hostInfo *HostInfo) {
	hm.Lock()
	defer hm.Unlock()
	hm.Hosts[hostInfo.Name] = hostInfo
}

type HostInfo struct {
	Remote  *udp.Addr
	Remotes *RemoteList
	//RemoteIndexId uint32
	//LocalIndexId  uint32
	Name string

	//RelayState RelayState

	// HandshakePacket 记录用于创建此主机信息的握手数据包
	// 我们需要这些数据包来避免重放握手数据包导致创建新的主机信息，从而引起不必要的变动
	//HandshakePacket map[uint8][]byte

	// LastHandshakeTime 记录远端在握手完成时告知我们的时间
	// 如果我是响应者，阶段1的数据包将包含此时间；如果我是发起者，则阶段2的数据包将包含此时间
	// 此字段用于避免在一段时间后重放握手数据包的攻击
	//LastHandshakeTime uint64

	// 用于跟踪此 VPN IP 的其他 HostInfo 对象，因为每个 VPN IP 只能有一个主要的 HostInfo 对象。
	// 通过 hostmap 锁进行同步，而不是通过 hostinfo 锁。
	//Next, Prev *HostInfo
}

func (h *HostInfo) SetRemote(remote *udp.Addr) {
	// 我们在这里复制是因为我们很可能从一个重用对象的源获取了这个 remote
	// 如果当前的 Remote 与传入的 remote 不相等，我们进行更新
	if !h.Remote.Equals(remote) {
		h.Remote = remote.Copy()
		h.Remotes.LearnRemote(h.Name, remote.Copy())
	}
}

func (h *HostInfo) String() string {
	marshal, err := json.Marshal(h)
	if err != nil {
		return ""
	}
	return string(marshal)
}

func NewRemoteList() *RemoteList {
	return &RemoteList{
		addrs: make([]*udp.Addr, 0),
		cache: make(map[string]*Cache),
	}
}
