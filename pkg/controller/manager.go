package controller

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"sync"
)

type Runnable interface {
	// Start 启动组件运行，当上下文关闭时，组件将停止运行。
	// Start 方法会阻塞，直到上下文关闭或发生错误。
	Start(context.Context) error
}

type Manager struct {
	runnables []Runnable
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	logger *zap.Logger
}

func NewManager(logger *zap.Logger, runnables ...Runnable) *Manager {
	return &Manager{
		runnables: runnables,
		logger:    logger,
	}
}

func (m *Manager) Start(ctx context.Context) error {
	ctx, m.cancel = context.WithCancel(ctx)

	for _, runnable := range m.runnables {
		m.wg.Add(1)
		go func(r Runnable) {
			defer m.wg.Done()
			if err := r.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				m.logger.Error("runnable error", zap.Error(err))
			}
		}(runnable)
	}

	m.wg.Wait()
	return nil
}

// Stop 停止所有 Runnable 组件
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}
