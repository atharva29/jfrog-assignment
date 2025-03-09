package pipeline

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type mockStage struct {
	process func(ctx context.Context, input interface{}) (interface{}, error)
}

func (m *mockStage) Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error {
	for item := range input {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			result, err := m.process(ctx, item)
			if err != nil {
				logger.Warn("mock stage process failed", zap.Error(err))
				continue
			}
			output <- result
		}
	}
	return nil
}

func TestPipeline_Execute(t *testing.T) {
	logger := zaptest.NewLogger(t)
	p := New(logger)

	// Stage 1: Multiply by 2
	p.AddStage(&mockStage{
		process: func(ctx context.Context, input interface{}) (interface{}, error) {
			if n, ok := input.(int); ok {
				return n * 2, nil
			}
			return nil, nil
		},
	})

	// Stage 2: Add 3
	p.AddStage(&mockStage{
		process: func(ctx context.Context, input interface{}) (interface{}, error) {
			if n, ok := input.(int); ok {
				return n + 3, nil
			}
			return nil, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan interface{}, 2)
	inputChan <- 5
	inputChan <- 10
	close(inputChan)

	done := make(chan error)
	go func() {
		done <- p.Run(ctx, inputChan)
	}()

	err := <-done
	if err != nil {
		t.Errorf("pipeline execution failed: %v", err)
	}

	// Note: We can't directly capture output here due to pipeline design,
	// but we verify it completes without hanging
}

func TestPipeline_Cancel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	p := New(logger)

	p.AddStage(&mockStage{
		process: func(ctx context.Context, input interface{}) (interface{}, error) {
			time.Sleep(100 * time.Millisecond) // Simulate work
			return input, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	inputChan := make(chan interface{}, 1)
	inputChan <- 1

	go func() {
		time.Sleep(10 * time.Millisecond) // Let pipeline start
		cancel()
	}()

	err := p.Run(ctx, inputChan)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	close(inputChan)
}
