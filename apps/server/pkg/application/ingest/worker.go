package ingest

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job represents an ingest job to be processed
type Job struct {
	ProjectID string
	Events    []IngestEvent
}

// Worker processes ingest jobs asynchronously
type Worker struct {
	processor *EventProcessor
	jobs      chan Job
	wg        sync.WaitGroup
	shutdown  chan struct{}
}

// NewWorker creates a new ingest worker
func NewWorker(processor *EventProcessor, bufferSize int) *Worker {
	return &Worker{
		processor: processor,
		jobs:      make(chan Job, bufferSize),
		shutdown:  make(chan struct{}),
	}
}

// Start begins processing jobs in background
func (w *Worker) Start(workers int) {
	for i := 0; i < workers; i++ {
		w.wg.Add(1)
		go w.run()
	}
	slog.Info("ingest worker started", "workers", workers, "buffer_size", cap(w.jobs))
}

// Enqueue adds a job to the queue
func (w *Worker) Enqueue(job Job) bool {
	select {
	case w.jobs <- job:
		return true
	default:
		slog.Warn("ingest queue full, dropping job", "project_id", job.ProjectID, "events", len(job.Events))
		return false
	}
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop(timeout time.Duration) {
	close(w.shutdown)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("ingest worker stopped gracefully")
	case <-time.After(timeout):
		slog.Warn("ingest worker shutdown timeout", "pending_jobs", len(w.jobs))
	}
}

// QueueSize returns current number of pending jobs
func (w *Worker) QueueSize() int {
	return len(w.jobs)
}

func (w *Worker) run() {
	defer w.wg.Done()

	for {
		select {
		case <-w.shutdown:
			w.drain()
			return
		case job := <-w.jobs:
			w.processJob(job)
		}
	}
}

func (w *Worker) drain() {
	for {
		select {
		case job := <-w.jobs:
			w.processJob(job)
		default:
			return
		}
	}
}

func (w *Worker) processJob(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := w.processor.ProcessEvents(ctx, job.ProjectID, job.Events); err != nil {
		slog.Error("failed to process ingest job",
			"project_id", job.ProjectID,
			"events", len(job.Events),
			"error", err,
		)
	}
}
