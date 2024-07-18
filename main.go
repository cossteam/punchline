package main

import (
	"context"
	"fmt"
	"github.com/cossteam/punchline/cmd"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := cmd.App.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	//raddr, err := net.ResolveUDPAddr("udp", server)
	//if err != nil {
	//	panic(err)
	//}
	//
	//coordinator := make([]*net.UDPAddr, 0)
	//if raddr != nil {
	//	coordinator = append(coordinator, raddr)
	//}
	//
	//fmt.Println("raddr => ", raddr)
	//
	//logger, err := log.SetupLogger(logLevel)
	//if err != nil {
	//	panic(err)
	//}
	//
	//outside, err := udp.NewGenericListener(logger, raddr.IP, int(listenPort))
	//if err != nil {
	//	logger.Fatal("failed to create listener", zap.Error(err))
	//}
	//
	//makeup, err := udp.DialMakeup(raddr.IP)
	//if err != nil {
	//	logger.Fatal("failed to dial makeup", zap.Error(err))
	//}
	//
	//var runnables []apiv1.Runnable
	//if amServer {
	//	runnables = append(runnables, controller.NewServer(logger, uint32(listenPort), raddr.IP.String(), outside))
	//} else {
	//	runnables = append(runnables, controller.NewClient(logger, uint32(listenPort), "t1", makeup, coordinator))
	//}
	//
	//ctx1 := SetupSignalHandler()
	//ctx, _ := context.WithCancel(ctx1)
	//for _, r := range runnables {
	//	go func(r apiv1.Runnable) {
	//		r.Start(ctx1)
	//	}(r)
	//}
	//
	//for {
	//	select {
	//	case <-ctx.Done():
	//		return
	//	}
	//}

	//sigChan := make(chan os.Signal, 1)
	//signal.Notify(sigChan, syscall.SIGTERM)
	//signal.Notify(sigChan, syscall.SIGINT)
	//
	//for {
	//	select {
	//	case <-ctx.Done():
	//		return
	//	case <-sigChan:
	//		cancel()
	//		return
	//	}
	//}

	//pub := publisher.NewInMemoryPublisher()
	//
	//// 添加主题
	//err := pub.AddTopic("example")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//// 订阅主题
	//subID, err := pub.Subscribe("example", func(msg *publisher.Message) error {
	//	fmt.Printf("Received message: %s\n", msg.Data)
	//	return nil
	//})
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//// 发布消息
	//msg := publisher.Message{
	//	Event: "exampleEvent",
	//	Data:  "Hello, World!",
	//}
	//err = pub.Publish("example", msg)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//// 发布消息
	//msg2 := publisher.Message{
	//	Event: "exampleEvent",
	//	Data:  "example msg 2",
	//}
	//err = pub.Publish("example", msg2)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//// 取消订阅
	//err = pub.Unsubscribe("example", subID)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//// 发布消息
	//msg3 := publisher.Message{
	//	Event: "exampleEvent",
	//	Data:  "example msg 3",
	//}
	//err = pub.Publish("example", msg3)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//time.Sleep(time.Second * 5)
}

var onlyOneSignalHandler = make(chan struct{})

// SetupSignalHandler registers for SIGTERM and SIGINT. A context is returned
// which is canceled on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupSignalHandler() context.Context {
	close(onlyOneSignalHandler) // panics when called twice

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		fmt.Println("ss")
		cancel()
		fmt.Println(11)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return ctx
}

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT}
