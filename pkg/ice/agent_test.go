package ice

import (
	"context"
	"testing"
	"time"

	"github.com/cossteam/punchline/pkg/signal"
	"github.com/pion/ice/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockSignalingClient struct {
	mock.Mock
}

func (m *MockSignalingClient) Unsubscribe(ctx context.Context, topic string) error {
	return nil
}

func (m *MockSignalingClient) Close() error {
	return nil
}

func (m *MockSignalingClient) Publish(ctx context.Context, msg *signal.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockSignalingClient) Subscribe(ctx context.Context, target string, handler func(*signal.Message) error) error {
	args := m.Called(ctx, target, handler)
	return args.Error(0)
}

func TestNewICEAgentWrapper(t *testing.T) {
	logger := zap.NewNop()
	client := new(MockSignalingClient)
	source := "source-peer"
	target := "target-peer"
	stunServer := []string{"stun:stun.l.google.com:19302"}

	peer, err := NewICEAgentWrapper(logger, client, stunServer, source, target)
	assert.NoError(t, err, "创建 ICE agent wrapper 时不应该出现错误")
	assert.NotNil(t, peer, "peer 不应该为 nil")
	assert.Equal(t, source, peer.source, "source 应该匹配")
	assert.Equal(t, target, peer.target, "target 应该匹配")
	assert.Equal(t, ConnectionStateClosed, peer.connectionState, "初始状态应该是 ConnectionStateClosed")
}

func TestPeer_Start(t *testing.T) {
	logger := zap.NewNop()
	client := new(MockSignalingClient)
	client.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	client.On("Subscribe", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	peer, _ := NewICEAgentWrapper(logger, client, []string{"stun:stun.l.google.com:19302"}, "source-peer", "target-peer")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(1 * time.Second)
		cancel() // 模拟上下文取消来关闭 agent
	}()

	err := peer.Start(ctx)
	assert.NoError(t, err, "启动 peer 时不应该出现错误")
	assert.Equal(t, ConnectionStateIdle, peer.connectionState, "启动后状态应该是 ConnectionStateIdle")
}

func TestPeer_Close(t *testing.T) {
	logger := zap.NewNop()
	client := new(MockSignalingClient)

	peer, _ := NewICEAgentWrapper(logger, client, []string{"stun:stun.l.google.com:19302"}, "source-peer", "target-peer")

	err := peer.Close()
	assert.NoError(t, err, "关闭 peer 时不应该出现错误")
	assert.Equal(t, ConnectionStateClosed, peer.connectionState, "关闭后状态应该是 ConnectionStateClosed")
}

func TestPeer_handleSignalingMessage(t *testing.T) {
	logger := zap.NewNop()
	client := new(MockSignalingClient)

	peer, _ := NewICEAgentWrapper(logger, client, []string{"stun:stun.l.google.com:19302"}, "source-peer", "target-peer")
	peer.connectionState = ConnectionStateIdle

	creds := &signal.Credentials{
		Ufrag: "remoteUfrag",
		Pwd:   "remotePwd",
	}
	msg := &signal.Message{
		Credentials: creds,
	}

	err := peer.handleSignalingMessage(msg)
	assert.NoError(t, err, "处理信令消息时不应该出现错误")
	assert.Equal(t, creds, peer.remoteCredentials, "处理后远程凭证应该被更新")
}

func TestPeer_onLocalCandidate(t *testing.T) {
	logger := zap.NewNop()
	client := new(MockSignalingClient)
	client.On("Publish", mock.Anything, mock.Anything).Return(nil)

	peer, _ := NewICEAgentWrapper(logger, client, []string{"stun:stun.l.google.com:19302"}, "source-peer", "target-peer")
	peer.connectionState = ConnectionStateConnecting

	candidate, _ := ice.NewCandidateHost(&ice.CandidateHostConfig{
		CandidateID: "candidate-id",
		Address:     "127.0.0.1",
		Port:        3478,
		Network:     "udp",
	})

	peer.onLocalCandidate(candidate)
	assert.Equal(t, ConnectionStateConnecting, peer.connectionState, "添加本地候选人后状态应该是 ConnectionStateConnecting")
	client.AssertCalled(t, "Publish", mock.Anything, mock.Anything)
}

func TestPeer_Restart(t *testing.T) {
	logger := zap.NewNop()
	client := new(MockSignalingClient)

	peer, _ := NewICEAgentWrapper(logger, client, []string{"stun:stun.l.google.com:19302"}, "source-peer", "target-peer")
	peer.connectionState = ConnectionStateIdle

	err := peer.Restart()
	assert.NoError(t, err, "重新启动 ICE 会话时不应该出现错误")
	assert.Equal(t, ConnectionStateRestarting, peer.connectionState, "重新启动后状态应该是 ConnectionStateRestarting")
}
