package main

import (
	"context"
	"time"

	"klein-harness/internal/runtime"

	"github.com/zeromicro/go-zero/core/logx"
	zeroservice "github.com/zeromicro/go-zero/core/service"
)

type daemonLoopRunner func(context.Context, string, time.Duration, runtime.RunOptions) error

func newDaemonLoopService(root string, interval time.Duration, options runtime.RunOptions) zeroservice.Service {
	return &daemonLoopService{
		root:     root,
		interval: interval,
		options:  options,
		runLoop:  runtime.LoopContext,
	}
}

func (s *daemonLoopService) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	runLoop := s.runLoop
	if runLoop == nil {
		runLoop = runtime.LoopContext
	}
	if err := runLoop(ctx, s.root, s.interval, s.options); err != nil {
		logx.Errorf("dashboard daemon loop stopped with error: %v", err)
	}
}

func (s *daemonLoopService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
