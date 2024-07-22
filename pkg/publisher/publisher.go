package publisher

import "context"

type Message struct {
	Topic string
	Data  []byte
}

// topicFunc 是一个函数类型，定义了一个用于过滤消息的函数。
// 这个函数接收一个消息值 v（类型为 interface{}），并根据消息的内容决定是否处理该消息。
// 返回值为 bool，表示是否处理这个消息。
//
// 当订阅者在接收到消息时，可以使用这个函数来筛选哪些消息需要被处理，哪些消息可以忽略。
// 例如，你可以根据消息的某些特定属性来决定是否将其传递给订阅者。
//
// 示例用法：
//
//	func myTopicFunc(v interface{}) bool {
//	    // 对消息进行检查，返回 true 表示处理消息，返回 false 表示忽略消息
//	    msg, ok := v.(MyMessageType)
//	    if !ok {
//	        return false
//	    }
//	    return msg.IsRelevant()
//	}
type topicFunc func(v interface{}) bool

type Publisher interface {
	// Len 返回当前订阅者数量。
	Len() int

	// Subscribe 添加一个新的订阅者，并返回一个频道供订阅者接收消息。
	Subscribe() chan interface{}

	// SubscribeTopic 添加一个新的订阅者，并根据指定的主题过滤消息。
	// 返回一个频道供订阅者接收消息。
	SubscribeTopic(topic topicFunc) chan interface{}

	// SubscribeTopicWithBuffer 添加一个新的订阅者，并根据指定的主题过滤消息。
	// 创建一个带有指定缓冲区大小的频道供订阅者接收消息。
	SubscribeTopicWithBuffer(topic topicFunc, buffer int) chan interface{}

	// Evict 移除指定的订阅者，使其不再接收任何消息。
	Evict(sub chan interface{})

	// Publish 向所有当前注册的订阅者发送数据。
	Publish(v interface{})

	// Close 关闭所有订阅者的频道。
	Close()
}

// PublisherClient 是用于发布和订阅消息的客户端接口
type PublisherClient interface {
	// Publish 将消息发布到指定的主题
	Publish(ctx context.Context, message *Message) error

	// Subscribe 订阅指定主题的消息，并提供一个处理函数来处理收到的消息
	Subscribe(ctx context.Context, topic string, handler func(*Message) error) error

	// Unsubscribe 取消订阅指定主题的消息
	Unsubscribe(ctx context.Context, topic string) error

	// Close 关闭客户端连接
	Close() error
}
