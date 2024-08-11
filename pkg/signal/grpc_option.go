package signal

type ClientOption interface {
	apply(*SignalingClient)
}

// WithClientName 返回一个设置客户端名称的选项
func WithClientName(name string) ClientOption {
	return clientOptionFunc(func(c *SignalingClient) {
		c.hostname = name
	})
}

// clientOptionFunc 是一个实现 ClientOption 接口的函数类型
type clientOptionFunc func(*SignalingClient)

func (f clientOptionFunc) apply(c *SignalingClient) {
	f(c)
}
