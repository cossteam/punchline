package publisher

import (
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

func TestPublisherPerformance(t *testing.T) {
	// 创建发布订阅实例
	pub := NewInMemoryPublisher()

	// 添加测试主题
	err := pub.AddTopic("testTopic")
	if err != nil {
		t.Fatalf("Failed to add topic: %v", err)
	}

	// 模拟订阅者处理函数
	handler := func(msg *Message) error {
		// 模拟处理消息的延迟
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	// 订阅测试主题
	numSubscribers := 1000 // 假设1000个订阅者
	var subscriberIDs []string
	for i := 0; i < numSubscribers; i++ {
		subID := uuid.New().String()
		err = pub.Subscribe("testTopic", subID, handler)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		subscriberIDs = append(subscriberIDs, subID)
	}

	// 测量发布消息的性能
	numMessages := 10000 // 假设发送10000条消息
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(numMessages)

	for i := 0; i < numMessages; i++ {
		go func(i int) {
			defer wg.Done()
			msg := Message{
				Event: "testEvent",
				Data:  i,
			}
			err := pub.Publish("testTopic", msg)
			if err != nil {
				t.Errorf("Failed to publish message: %v", err)
			}
		}(i)
	}

	wg.Wait()

	elapsed := time.Since(start)
	tps := float64(numMessages) / elapsed.Seconds()
	t.Logf("Published %d messages in %.2f seconds. Throughput: %.2f messages/second", numMessages, elapsed.Seconds(), tps)

	// 取消订阅所有订阅者
	for _, subID := range subscriberIDs {
		err := pub.Unsubscribe("testTopic", subID)
		if err != nil {
			t.Errorf("Failed to unsubscribe: %v", err)
		}
	}
}
