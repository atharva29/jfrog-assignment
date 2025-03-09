package pipeline

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Stage defines the interface for a pipeline stage.
// Each stage processes input from an input channel and sends results to an output channel.
type Stage interface {
	Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error
}

// Pipeline manages a sequence of stages that process data in a chain.
type Pipeline struct {
	stages []Stage     // List of stages in the pipeline
	logger *zap.Logger // Logger for pipeline-wide logging
}

// New creates a new Pipeline instance with the given logger.
//
// Parameters:
//   - logger: Logger for logging pipeline events.
//
// Returns:
//   - A pointer to a new Pipeline instance.
func New(logger *zap.Logger) *Pipeline {
	return &Pipeline{
		logger: logger,
	}
}

// AddStage adds a stage to the pipeline's sequence.
//
// Parameters:
//   - stage: The stage to add.
func (p *Pipeline) AddStage(stage Stage) {
	p.stages = append(p.stages, stage)
}

// Run executes the pipeline with the given input channel.
//
// The pipeline chains stages such that each stage's output becomes the next stage's input.
// The first stage uses the provided input channel, and subsequent stages use channels created internally.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts.
//   - input: Initial input channel for the first stage.
//
// Returns:
//   - An error if the pipeline fails to complete (e.g., due to cancellation), nil otherwise.
func (p *Pipeline) Run(ctx context.Context, input <-chan interface{}) error {
	if len(p.stages) == 0 {
		p.logger.Warn("no stages in pipeline")
		return nil
	}

	channels := make([]chan interface{}, len(p.stages))
	for i := range channels {
		channels[i] = make(chan interface{}, 50)
	}

	var wg sync.WaitGroup
	wg.Add(len(p.stages))

	for i, stage := range p.stages {
		inChan := input
		if i > 0 {
			inChan = channels[i-1]
		}
		outChan := channels[i]

		go func(stage Stage, in <-chan interface{}, out chan<- interface{}, idx int) {
			defer wg.Done()
			defer close(out)
			if err := stage.Execute(ctx, in, out, p.logger); err != nil {
				p.logger.Error("stage execution failed",
					zap.Int("stage", idx),
					zap.Error(err))
			}
		}(stage, inChan, outChan, i)
	}

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
