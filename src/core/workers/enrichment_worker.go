// ABOUTME: Enrichment worker handles background processing of metadata and color extraction
// ABOUTME: Provides managed worker pools for asynchronous content enrichment

package workers

import (
	"context"
	"sync"
	"time"
	
	"digests-app-api/core/interfaces"
)

// EnrichmentJob represents a job for enrichment processing
type EnrichmentJob struct {
	Type      JobType
	URLs      []string
	Context   context.Context
	ResultCh  chan<- interface{}
	ErrorCh   chan<- error
}

// JobType represents the type of enrichment job
type JobType int

const (
	JobTypeMetadata JobType = iota
	JobTypeColor
)

// EnrichmentWorker manages background enrichment processing
type EnrichmentWorker struct {
	enrichmentService interfaces.ContentEnrichmentService
	jobQueue         chan *EnrichmentJob
	maxWorkers       int
	queueSize        int
	workers          []*worker
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.Mutex
	running          bool
}

// worker represents an individual worker goroutine
type worker struct {
	id               int
	jobQueue         <-chan *EnrichmentJob
	enrichmentService interfaces.ContentEnrichmentService
	ctx              context.Context
	wg               *sync.WaitGroup
}

// WorkerConfig holds configuration for the enrichment worker
type WorkerConfig struct {
	MaxWorkers int
	QueueSize  int
}

// DefaultWorkerConfig returns the default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		MaxWorkers: 10,
		QueueSize:  100,
	}
}

// NewEnrichmentWorker creates a new enrichment worker
func NewEnrichmentWorker(enrichmentService interfaces.ContentEnrichmentService, config WorkerConfig) *EnrichmentWorker {
	ctx, cancel := context.WithCancel(context.Background())
	
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = DefaultWorkerConfig().MaxWorkers
	}
	if config.QueueSize <= 0 {
		config.QueueSize = DefaultWorkerConfig().QueueSize
	}
	
	return &EnrichmentWorker{
		enrichmentService: enrichmentService,
		jobQueue:         make(chan *EnrichmentJob, config.QueueSize),
		maxWorkers:       config.MaxWorkers,
		queueSize:        config.QueueSize,
		workers:          make([]*worker, 0, config.MaxWorkers),
		ctx:              ctx,
		cancel:           cancel,
		running:          false,
	}
}

// Start starts the worker pool
func (ew *EnrichmentWorker) Start() error {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	
	if ew.running {
		return nil
	}
	
	// Create and start workers
	for i := 0; i < ew.maxWorkers; i++ {
		w := &worker{
			id:                i,
			jobQueue:          ew.jobQueue,
			enrichmentService: ew.enrichmentService,
			ctx:               ew.ctx,
			wg:                &ew.wg,
		}
		ew.workers = append(ew.workers, w)
		ew.wg.Add(1)
		go w.run()
	}
	
	ew.running = true
	return nil
}

// Stop stops the worker pool gracefully
func (ew *EnrichmentWorker) Stop() error {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	
	if !ew.running {
		return nil
	}
	
	// Cancel context to signal workers to stop
	ew.cancel()
	
	// Close job queue
	close(ew.jobQueue)
	
	// Wait for all workers to finish
	ew.wg.Wait()
	
	ew.running = false
	return nil
}

// SubmitJob submits a job to the worker pool
func (ew *EnrichmentWorker) SubmitJob(job *EnrichmentJob) error {
	ew.mu.Lock()
	if !ew.running {
		ew.mu.Unlock()
		return ErrWorkerNotRunning
	}
	ew.mu.Unlock()
	
	select {
	case ew.jobQueue <- job:
		return nil
	case <-time.After(5 * time.Second):
		return ErrQueueFull
	}
}

// ExtractColorBatch submits a batch color extraction job
func (ew *EnrichmentWorker) ExtractColorBatch(ctx context.Context, imageURLs []string) {
	job := &EnrichmentJob{
		Type:    JobTypeColor,
		URLs:    imageURLs,
		Context: ctx,
	}
	
	// Submit job and ignore errors for backward compatibility
	_ = ew.SubmitJob(job)
}

// ExtractMetadataBatch submits a batch metadata extraction job
func (ew *EnrichmentWorker) ExtractMetadataBatch(ctx context.Context, urls []string) {
	job := &EnrichmentJob{
		Type:    JobTypeMetadata,
		URLs:    urls,
		Context: ctx,
	}
	
	// Submit job and ignore errors for backward compatibility
	_ = ew.SubmitJob(job)
}

// run is the main loop for each worker
func (w *worker) run() {
	defer w.wg.Done()
	
	for {
		select {
		case job, ok := <-w.jobQueue:
			if !ok {
				// Channel closed, exit
				return
			}
			w.processJob(job)
		case <-w.ctx.Done():
			// Context cancelled, exit
			return
		}
	}
}

// processJob processes a single enrichment job
func (w *worker) processJob(job *EnrichmentJob) {
	switch job.Type {
	case JobTypeColor:
		// Process color extraction
		results := w.enrichmentService.ExtractColorBatch(job.Context, job.URLs)
		if job.ResultCh != nil {
			select {
			case job.ResultCh <- results:
			case <-job.Context.Done():
			}
		}
		
	case JobTypeMetadata:
		// Process metadata extraction
		results := w.enrichmentService.ExtractMetadataBatch(job.Context, job.URLs)
		if job.ResultCh != nil {
			select {
			case job.ResultCh <- results:
			case <-job.Context.Done():
			}
		}
	}
}

// Error definitions
var (
	ErrWorkerNotRunning = &WorkerError{Message: "worker pool is not running"}
	ErrQueueFull        = &WorkerError{Message: "job queue is full"}
)

// WorkerError represents a worker-specific error
type WorkerError struct {
	Message string
}

func (e *WorkerError) Error() string {
	return e.Message
}