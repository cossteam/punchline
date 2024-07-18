package publisher

type Event string

type Message struct {
	Event Event
	Data  interface{}
}

type Publisher interface {
	// AddTopic 添加新的主题
	AddTopic(name string) error

	// Publish 发布消息到指定主题
	Publish(topic string, message Message) error

	// Subscribe 订阅指定主题，返回订阅者ID以便于取消订阅
	Subscribe(topic string, clientID string, handler func(*Message) error) error

	// Unsubscribe 通过订阅者ID取消订阅
	Unsubscribe(topic string, subscriberID string) error
}

type Client interface {
	Close() error

	// AddTopic 添加新的主题
	AddTopic(name string) error

	// Publish 发布消息到指定主题
	Publish(topic string, message Message) error

	// Subscribe 订阅指定主题的消息
	Subscribe(topic string, handler func(*Message) error) error

	// Unsubscribe 取消订阅指定主题的消息
	Unsubscribe(topic string) error
}
