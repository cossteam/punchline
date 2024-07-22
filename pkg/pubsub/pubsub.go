package pubsub

// Message represents a message with a topic, event, and data.
type Message struct {
	Topic string
	Event string
	Data  string
}

// PublishRequest represents a request to publish a message.
type PublishRequest struct {
	Name string
	Data string
}

// PublishResponse represents a response to a publish request.
type PublishResponse struct{}

// SubscribeRequest represents a request to subscribe to messages.
type SubscribeRequest struct {
	Name string
}

// PubSubService defines the methods for a publish-subscribe service.
type PubSubService interface {
	Publish(request *PublishRequest) (*PublishResponse, error)
	Subscribe(request *SubscribeRequest) (<-chan *Message, error)
}
