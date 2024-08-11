package ice

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/cossteam/punchline/api/signaling/v1"
	"github.com/cossteam/punchline/pkg/signal"
	"github.com/pion/ice/v2"
	"github.com/pion/randutil"
	"github.com/pion/stun"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
	"sync/atomic"
	"time"
)

var (
	_iceURLs = []string{"stun:stun3.l.google.com:19302", "stun:stun.cunicu.li:3478", "stun:stun.easyvoip.com:3478"}

	errStillIdle                        = errors.New("not connected yet")
	errCreateNonClosedAgent             = errors.New("failed to create new agent if previous one is not closed")
	errSwitchToIdle                     = errors.New("failed to switch to idle state")
	errInvalidConnectionStateForRestart = errors.New("can not restart agent while in state")
)

// Peer 封装了 Pion ICE Agent 以简化点对点连接的建立
type Peer struct {
	logger *zap.Logger

	client signal.Client

	source   string
	target   string
	restarts atomic.Uint32

	connectionState   ConnectionState
	agent             *ice.Agent
	remoteCredentials *signaling.Credentials
	localCredentials  *signaling.Credentials
}

// NewICEAgentWrapper 创建并返回一个新的 Peer
func NewICEAgentWrapper(
	logger *zap.Logger,
	signalingClient signal.Client,
	source string,
	target string,
) (*Peer, error) {
	iceURLs, err := convertToStunURIs(_iceURLs)
	if err != nil {
		return nil, err
	}

	// 创建 ICE 配置
	iceConfig := ice.AgentConfig{
		Urls: iceURLs,
		NetworkTypes: []ice.NetworkType{
			ice.NetworkTypeTCP4,
			ice.NetworkTypeTCP6,
			ice.NetworkTypeUDP4,
			ice.NetworkTypeUDP6,
		},
		CandidateTypes: []ice.CandidateType{
			ice.CandidateTypeHost,
			ice.CandidateTypeServerReflexive,
			ice.CandidateTypePeerReflexive,
			//ice.CandidateTypeRelay,
		},
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

	fmt.Println("localUfrag => ", localUfrag)

	wrapper := &Peer{
		logger: logger,
		client: signalingClient,
		source: source,
		target: target,

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

func (p *Peer) Start(ctx context.Context) error {
	serverShutdown := make(chan struct{})
	go func() {
		<-ctx.Done()
		p.logger.Info("Shutting down SignalingServer")
		if err := p.agent.Close(); err != nil {
			p.logger.Error("Failed to close agent", zap.Error(err))
		}
		close(serverShutdown)
	}()

	if p.connectionState != ConnectionStateClosed {
		return errCreateNonClosedAgent
	} else {
		p.connectionState = ConnectionStateCreating
	}

	// Reset state to ConnectionStateCreating if there is an error later
	defer func() {
		if p.connectionState == ConnectionStateCreating {
			p.connectionState = ConnectionStateClosed
		}
	}()

	//
	//if err := p.client.Publish(ctx, &signal.Message{
	//	Topic: p.target,
	//}); err != nil {
	//	return err
	//}

	if err := p.client.Subscribe(ctx, p.target, p.handleSignalingMessage); err != nil {
		return err
	}

	// 当我们收集到一个新的ICE候选对象时，将其发送到远程对等方
	if err := p.agent.OnCandidate(p.onLocalCandidate); err != nil {
		return fmt.Errorf("failed to set candidate callback: %v", err)
	}

	// When selected candidate pair changes
	if err := p.agent.OnSelectedCandidatePairChange(p.onSelectedCandidatePairChange); err != nil {
		return fmt.Errorf("failed to setup on selected candidate pair handler: %w", err)
	}

	// When ICE Connection state has change print to stdout
	if err := p.agent.OnConnectionStateChange(p.onConnectionStateChange); err != nil {
		return err
	}

	currentState := p.connectionState

	// 检查当前状态是否为 ConnectionStateCreating
	if currentState == ConnectionStateCreating {
		p.connectionState = ConnectionStateIdle
	} else {
		return errSwitchToIdle
	}

	// 开始连接检查
	if err := p.agent.GatherCandidates(); err != nil {
		return fmt.Errorf("failed to gather candidates: %v", err)
	}

	// 设置远程用户名片段和密码
	//if err := p.agent.SetRemoteCredentials(p.remoteUfrag, p.remotePwd); err != nil {
	//	return fmt.Errorf("failed to set remote credentials: %v", err)
	//}

	// Send peer credentials as long as we remain in ConnectionStateIdle
	go p.sendCredentialsWhileIdleWithBackoff(true)

	<-serverShutdown

	return nil
}

func (p *Peer) connect(ufrag, pwd string) {
	var connect func(context.Context, string, string) (*ice.Conn, error)
	if p.IsControlling() {
		p.logger.Debug("Dialing...")
		connect = p.agent.Dial
	} else {
		p.logger.Debug("Accepting...")
		connect = p.agent.Accept
	}

	conn, err := connect(context.TODO(), ufrag, pwd)
	if err != nil {
		p.logger.Error("Failed to connect", zap.Error(err))
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
func (p *Peer) Close() error {
	return p.agent.Close()
}

func (p *Peer) onLocalCandidate(c ice.Candidate) {
	if c == nil {
		return
	}

	logger := p.logger.With(zap.Stringer("candidate", c))
	logger.Debug("Added local candidate to agent", zap.Stringer("state", p.connectionState))

	if err := p.sendCandidate(c); err != nil {
		logger.Error("Failed to send candidate", zap.Error(err))
	}

	if p.connectionState == ConnectionStateGatheringLocal {
		p.connectionState = ConnectionStateConnecting
		go p.connect(p.remoteCredentials.Ufrag, p.remoteCredentials.Pwd)
	} else if p.connectionState == ConnectionStateGathering {
		// Continue waiting until we received the first remote candidate
		p.connectionState = ConnectionStateGatheringRemote
	}
}

func (p *Peer) sendCandidate(c ice.Candidate) error {
	msg := &signaling.Message{
		Candidate: signaling.NewCandidate(c),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// TODO 发送到信令服务器
	if err := p.client.Publish(ctx, &signal.Message{
		Topic:     p.source,
		Data:      nil,
		Candidate: msg.Candidate,
	}); err != nil {
		return err
	}

	p.logger.Debug("Sent candidate", zap.Reflect("candidate", msg.Candidate))

	return nil
}

func (p *Peer) onConnectionStateChange(state ice.ConnectionState) {
	cs := signaling.NewConnectionState(state)

	p.logger.Debug("ICE connection state changed", zap.Reflect("state", cs))

	switch cs {
	case ConnectionStateFailed, ConnectionStateDisconnected:

	case ConnectionStateClosed:

	case ConnectionStateConnected:

	default:
	}
}

func (p *Peer) handleSignalingMessage(message *signal.Message) error {
	p.logger.Debug("Received signaling message",
		zap.Stringer("state", p.connectionState),
		zap.Any("credentials", message.Credentials),
		zap.Any("message", message))

	if message.Credentials != nil {
		p.onRemoteCredentials(message.Credentials)
	}

	if message.Candidate != nil {
		p.onRemoteCandidate(message.Candidate)
	}

	return nil
}

// onRemoteCredentials is a handler called for each received pair of remote Ufrag/Pwd via the signaling channel
func (p *Peer) onRemoteCredentials(creds *signal.Credentials) {
	logger := p.logger.With(zap.Reflect("creds", creds))
	logger.Debug("Received remote credentials", zap.Stringer("state", p.connectionState))

	if p.isSessionRestart(creds) {
		if err := p.Restart(); err != nil {
			p.logger.Error("Failed to restart ICE session", zap.Error(err))
		}
	} else {
		if p.connectionState != ConnectionStateIdle {
			p.logger.Debug("Ignoring duplicated credentials")
			return
		}
		// 如果当前状态为 ConnectionStateIdle，更新为 ConnectionStateGathering
		p.connectionState = ConnectionStateGathering

		//p.SetStateIf(daemon.PeerStateConnecting, daemon.PeerStateClosed, daemon.PeerStateFailed, daemon.PeerStateNew)

		p.remoteCredentials = creds

		// Return our own credentials if requested
		if creds.NeedCreds {
			if err := p.sendCredentials(false); err != nil {
				p.logger.Error("Failed to send credentials", zap.Error(err))
				return
			}
		}

		// Start gathering candidates
		if err := p.agent.GatherCandidates(); err != nil {
			p.logger.Error("failed to gather candidates", zap.Error(err))
			return
		}
	}
}

func (p *Peer) sendCredentials(need bool) error {
	p.localCredentials.NeedCreds = need

	msg := &signaling.Message{
		Topic: p.source,

		Credentials: p.localCredentials,
	}

	// TODO: Is this timeout suitable?
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := p.client.Publish(ctx, &signal.Message{
		Topic:       p.source,
		Data:        nil,
		Credentials: msg.Credentials,
	}); err != nil {
		return err
	}

	p.logger.Debug("Sent credentials", zap.Reflect("creds", msg.Credentials))

	return nil
}

// Restart the ICE agent by creating a new one
func (p *Peer) Restart() error {
	if p.connectionState == ConnectionStateClosed || p.connectionState == ConnectionStateClosing || p.connectionState == ConnectionStateRestarting {
		return fmt.Errorf("%w: %s", errInvalidConnectionStateForRestart, strings.ToLower(p.connectionState.String()))
	}

	p.connectionState = ConnectionStateRestarting
	p.logger.Debug("Restarting ICE session")

	if err := p.agent.Close(); err != nil {
		return fmt.Errorf("failed to close agent: %w", err)
	}

	// The new agent will be recreated in the onConnectionStateChange() handler
	// once the old agent has been properly closed

	p.restarts.Add(1)

	return nil
}

// isSessionRestart checks if a received offer should restart the
// ICE session by comparing ufrag & pwd with previously used values.
func (p *Peer) isSessionRestart(c *signal.Credentials) bool {
	r := p.remoteCredentials
	return (r != nil) &&
		(r.Ufrag != "" && r.Pwd != "") &&
		(c.Ufrag != "" && c.Pwd != "") &&
		(r.Ufrag != c.Ufrag || r.Pwd != c.Pwd)
}

// AddRemoteCandidate 添加远程候选者
func (p *Peer) AddRemoteCandidate(c *ice.Candidate) error {
	return p.agent.AddRemoteCandidate(*c)
}

func (p *Peer) IsControlling() bool {
	var pkOur, pkTheir big.Int
	pkOur.SetBytes([]byte(p.source))
	pkTheir.SetBytes([]byte(p.target))

	return pkOur.Cmp(&pkTheir) == -1
}

// onRemoteCandidate is a handler called for each received candidate via the signaling channel
func (p *Peer) onRemoteCandidate(c *signal.Candidate) {
	logger := p.logger.With(zap.Reflect("candidate", c))

	ic, err := c.ICECandidate()
	if err != nil {
		logger.Error("Failed to remote candidate", zap.Error(err))
		return
	}

	if err := p.agent.AddRemoteCandidate(ic); err != nil {
		logger.Error("Failed to add remote candidate", zap.Error(err))
		return
	}

	logger.Debug("Added remote candidate to agent", zap.Stringer("state", p.connectionState))

	if p.connectionState == ConnectionStateGatheringRemote {
		p.connectionState = ConnectionStateConnecting
		go p.connect(p.remoteCredentials.Ufrag, p.remoteCredentials.Pwd)
	} else if p.connectionState == ConnectionStateGathering {
		p.connectionState = ConnectionStateGatheringLocal
	}
}

func (p *Peer) onSelectedCandidatePairChange(local ice.Candidate, remote ice.Candidate) {
	p.logger.Info("Selected new candidate pair",
		zap.Any("local", local),
		zap.Any("remote", remote),
	)
}

func (p *Peer) sendCredentialsWhileIdleWithBackoff(need bool) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 1 * time.Minute

	if err := backoff.RetryNotify(
		func() error {
			if p.connectionState != ConnectionStateIdle {
				// We are not idling any more.
				// No need to send credentials
				return nil
			}

			if err := p.sendCredentials(need); err != nil {
				if status.Code(err) == codes.Canceled {
					// Do not retry when the signaling backend has been closed
					return nil
				}

				return err
			}

			return errStillIdle
		}, bo,
		func(err error, d time.Duration) {
			if errors.Is(err, errStillIdle) {
				p.logger.Debug("Sending peer credentials while waiting for remote peer",
					zap.Error(err),
					zap.Duration("after", d))
			} else if sts := status.Code(err); sts != codes.Canceled {
				p.logger.Error("Failed to send peer credentials",
					zap.Error(err),
					zap.Duration("after", d))
			}
		},
	); err != nil {
		p.logger.Error("Failed to send credentials", zap.Error(err))
	}
}

func convertToStunURIs(urls []string) ([]*stun.URI, error) {
	var iceURLs []*stun.URI
	for _, url := range urls {
		uri, err := stun.ParseURI(url)
		if err != nil {
			return nil, err
		}
		iceURLs = append(iceURLs, uri)
	}
	return iceURLs, nil
}
