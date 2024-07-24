package api

import "context"

type EncWriter interface {
	//SendTo(t header.MessageType, st header.MessageSubType, vpnIp api.VpnIp, p, out []byte)
}

type Runnable interface {
	// Start 启动组件运行，当上下文关闭时，组件将停止运行。
	// Start 方法会阻塞，直到上下文关闭或发生错误。
	Start(context.Context) error
}
