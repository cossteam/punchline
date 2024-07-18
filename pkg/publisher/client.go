package publisher

var _ Client = &client{}

type client struct {
	clientId  string
	publisher Publisher
}

func (c *client) Close() error {
	//TODO implement me
	panic("implement me")
}

func (c *client) AddTopic(name string) error {
	return c.publisher.AddTopic(name)
}

func (c *client) Publish(topic string, message Message) error {
	return c.publisher.Publish(topic, message)
}

func (c *client) Subscribe(topic string, handler func(*Message) error) error {
	return c.publisher.Subscribe(topic, c.clientId, handler)
}

func (c *client) Unsubscribe(topic string) error {
	return c.publisher.Unsubscribe(topic, c.clientId)
}
