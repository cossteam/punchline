package signal

import "context"

type Client interface {
	// Publish 将消息发布到指定的主题
	Publish(ctx context.Context, message *Message) error

	// Subscribe 订阅指定主题的消息，并提供一个处理函数来处理收到的消息
	Subscribe(ctx context.Context, topic string, handler func(*Message) error) error

	// Unsubscribe 取消订阅指定主题的消息
	Unsubscribe(ctx context.Context, topic string) error

	// Close 关闭客户端连接
	Close() error
}
