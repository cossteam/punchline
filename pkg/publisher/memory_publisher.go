package publisher

import (
	"errors"
	"log"
	"sync"
)

type inMemoryPublisher struct {
	mu     sync.RWMutex
	topics map[string]*topicData
}

type topicData struct {
	mu          sync.RWMutex
	subscribers map[string]func(*Message) error
}

func NewInMemoryPublisher() Publisher {
	return &inMemoryPublisher{
		topics: make(map[string]*topicData),
	}
}

func (p *inMemoryPublisher) AddTopic(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.topics[name]; exists {
		return errors.New("topic already exists")
	}
	p.topics[name] = &topicData{
		subscribers: make(map[string]func(*Message) error),
	}
	return nil
}

func (p *inMemoryPublisher) Publish(topic string, message Message) error {
	p.mu.RLock()
	td, exists := p.topics[topic]
	if !exists {
		p.mu.RUnlock()
		return errors.New("topic not found")
	}

	td.mu.RLock()
	p.mu.RUnlock()

	for _, handler := range td.subscribers {
		go func(handler func(*Message) error) {
			if err := handler(&message); err != nil {
				log.Printf("Error handling message for topic %s: %v", topic, err)
			}
		}(handler)
	}

	td.mu.RUnlock()
	return nil
}

func (p *inMemoryPublisher) Subscribe(topic, subscriberID string, handler func(*Message) error) error {
	p.mu.Lock()
	td, exists := p.topics[topic]
	if !exists {
		p.mu.Unlock()
		return errors.New("topic not found")
	}

	td.mu.Lock()
	p.mu.Unlock()
	td.subscribers[subscriberID] = handler
	td.mu.Unlock()
	return nil
}

func (p *inMemoryPublisher) Unsubscribe(topic string, subscriberID string) error {
	p.mu.Lock()
	td, exists := p.topics[topic]
	if !exists {
		p.mu.Unlock()
		return errors.New("topic not found")
	}

	td.mu.Lock()
	p.mu.Unlock()

	if _, exists := td.subscribers[subscriberID]; !exists {
		td.mu.Unlock()
		return errors.New("subscriber not found")
	}
	delete(td.subscribers, subscriberID)
	td.mu.Unlock()
	return nil
}
