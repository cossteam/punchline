package ice

import (
	"context"
	"fmt"
	"github.com/cossteam/punchline/api/signaling/v1"
	"github.com/pion/ice/v2"
	"github.com/pion/randutil"
	"go.uber.org/zap"
	"math/big"
	"time"
)

// ICEAgentWrapper 封装了 Pion ICE Agent 以简化点对点连接的建立
type ICEAgentWrapper struct {
	logger *zap.Logger

	signalingClient signaling.SignalingClient

	source string
	target string
	//intf *wgtypes.Device
	//peer *wgtypes.Peer

	connectionState   ConnectionState
	agent             *ice.Agent
	remoteCredentials *signaling.Credentials
	localCredentials  *signaling.Credentials
}

// NewICEAgentWrapper 创建并返回一个新的 ICEAgentWrapper
func NewICEAgentWrapper(
	logger *zap.Logger,
	signalingClient signaling.SignalingClient,
	source string,
	target string,
) (*ICEAgentWrapper, error) {
	// 创建 ICE 配置
	iceConfig := ice.AgentConfig{
		NetworkTypes: []ice.NetworkType{ice.NetworkTypeUDP4, ice.NetworkTypeUDP6},
	}

	// 创建新的 ICE Agent
	agent, err := ice.NewAgent(&iceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ICE agent: %v", err)
	}

	// 获取本地用户名片段和密码
	localUfrag, localPwd, err := agent.GetLocalUserCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get local user credentials: %v", err)
	}

	wrapper := &ICEAgentWrapper{
		logger:          logger,
		signalingClient: signalingClient,
		source:          source,
		target:          target,

		agent:           agent,
		connectionState: ConnectionStateClosed,
		localCredentials: &signaling.Credentials{
			Ufrag:     localUfrag,
			Pwd:       localPwd,
			NeedCreds: false,
		},
	}

	return wrapper, nil
}

// AddRemoteCandidate 添加远程候选者
func (w *ICEAgentWrapper) AddRemoteCandidate(c *ice.Candidate) error {
	return w.agent.AddRemoteCandidate(*c)
}

func (w *ICEAgentWrapper) Start(ctx context.Context) error {
	// 当我们收集到一个新的ICE候选对象时，将其发送到远程对等方
	if err := w.agent.OnCandidate(w.onLocalCandidate); err != nil {
		return fmt.Errorf("failed to set candidate callback: %v", err)
	}

	// When ICE Connection state has change print to stdout
	if err := w.agent.OnConnectionStateChange(w.onConnectionStateChange); err != nil {
		return err
	}

	// 开始连接检查
	if err := w.agent.GatherCandidates(); err != nil {
		return fmt.Errorf("failed to gather candidates: %v", err)
	}

	w.connect(w.localCredentials.Ufrag, w.localCredentials.Pwd)

	// 设置远程用户名片段和密码
	//if err := w.agent.SetRemoteCredentials(w.remoteUfrag, w.remotePwd); err != nil {
	//	return fmt.Errorf("failed to set remote credentials: %v", err)
	//}

	return nil
}

func (w *ICEAgentWrapper) connect(ufrag, pwd string) {
	var connect func(context.Context, string, string) (*ice.Conn, error)
	if w.IsControlling() {
		w.logger.Debug("Dialing...")
		connect = w.agent.Dial
	} else {
		w.logger.Debug("Accepting...")
		connect = w.agent.Accept
	}

	conn, err := connect(context.TODO(), ufrag, pwd)
	if err != nil {
		w.logger.Error("Failed to connect", zap.Error(err))
		return
	}

	// Send messages in a loop to the remote peer
	go func() {
		for {
			time.Sleep(time.Second * 3)

			val, err := randutil.GenerateCryptoRandomString(15, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
			if err != nil {
				panic(err)
			}
			if _, err = conn.Write([]byte(val)); err != nil {
				panic(err)
			}

			fmt.Printf("Sent: '%s'\n", val)
		}
	}()

	// Receive messages in a loop from the remote peer
	buf := make([]byte, 1500)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Received: '%s'\n", string(buf[:n]))
	}
}

// Close 关闭 ICE Agent
func (w *ICEAgentWrapper) Close() error {
	return w.agent.Close()
}

func (w *ICEAgentWrapper) onLocalCandidate(c ice.Candidate) {
	if c == nil {
		return
	}

	logger := w.logger.With(zap.Reflect("candidate", c))
	logger.Debug("Added local candidate to agent")

	if err := w.sendCandidate(c); err != nil {
		logger.Error("Failed to send candidate", zap.Error(err))
	}

	//if w.connectionState == ConnectionStateGatheringLocal {
	//	w.connectionState = ConnectionStateConnecting
	//	go w.connect(w.remoteCredentials.Ufrag, w.remoteCredentials.Pwd)
	//} else if w.connectionState == ConnectionStateGathering {
	//	// Continue waiting until we received the first remote candidate
	//	w.connectionState = ConnectionStateGatheringRemote
	//}
}

func (w *ICEAgentWrapper) sendCandidate(c ice.Candidate) error {
	msg := &signaling.Message{
		Candidate: signaling.NewCandidate(c),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// TODO 发送到信令服务器
	_, err := w.signalingClient.Publish(ctx, &signaling.PublishRequest{
		Topic:    w.source,
		Hostname: w.source,
		Data:     nil,

		Candidate: msg.Candidate,
	})
	if err != nil {
		return err
	}

	w.logger.Debug("Sent candidate", zap.Reflect("candidate", msg.Candidate))

	return nil
}

func (w *ICEAgentWrapper) onConnectionStateChange(state ice.ConnectionState) {
	cs := signaling.NewConnectionState(state)

	w.logger.Debug("ICE connection state changed", zap.Reflect("state", cs))
}

func (w *ICEAgentWrapper) IsControlling() bool {
	var pkOur, pkTheir big.Int
	pkOur.SetBytes([]byte(w.source))
	pkTheir.SetBytes([]byte(w.target))

	return pkOur.Cmp(&pkTheir) == -1
}
