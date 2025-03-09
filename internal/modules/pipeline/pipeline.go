package pipeline

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Stage defines the interface for a pipeline stage
type Stage interface {
	Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error
}

// Pipeline manages a sequence of stages
type Pipeline struct {
	stages []Stage
	logger *zap.Logger
}

// New creates a new Pipeline with the given logger
func New(logger *zap.Logger) *Pipeline {
	return &Pipeline{
		logger: logger,
	}
}

// AddStage adds a stage to the pipeline
func (p *Pipeline) AddStage(stage Stage) {
	p.stages = append(p.stages, stage)
}

// Run executes the pipeline with the given input channel
func (p *Pipeline) Run(ctx context.Context, input <-chan interface{}) error {
	if len(p.stages) == 0 {
		p.logger.Warn("no stages in pipeline")
		return nil
	}

	// Create channels for each stage transition
	channels := make([]chan interface{}, len(p.stages))
	for i := range channels {
		channels[i] = make(chan interface{}, 50)
	}

	var wg sync.WaitGroup
	wg.Add(len(p.stages))

	// Start each stage
	for i, stage := range p.stages {
		inChan := input
		if i > 0 {
			inChan = channels[i-1] // Use previous stage's output as input
		}
		outChan := channels[i]

		go func(stage Stage, in <-chan interface{}, out chan<- interface{}, idx int) {
			defer wg.Done()
			defer close(out) // Ensure output channel is closed when stage is done
			if err := stage.Execute(ctx, in, out, p.logger); err != nil {
				p.logger.Error("stage execution failed",
					zap.Int("stage", idx),
					zap.Error(err))
			}
		}(stage, inChan, outChan, i)
	}

	// Wait for all stages to complete or context to be canceled
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("pipeline completed successfully")
		return nil
	case <-ctx.Done():
		p.logger.Info("pipeline canceled", zap.Error(ctx.Err()))
		return ctx.Err()
	}
}
