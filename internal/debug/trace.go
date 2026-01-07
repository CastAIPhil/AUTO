package debug

import (
	"fmt"
	"os"
	"runtime/trace"
	"sync"
)

// Tracer manages execution tracing to a file
type Tracer struct {
	file *os.File
	mu   sync.Mutex
}

// NewTracer creates a new tracer
func NewTracer() *Tracer {
	return &Tracer{}
}

// Start begins tracing to the specified file.
// Analyze the output with: go tool trace <filename>
func (t *Tracer) Start(filename string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.file != nil {
		return fmt.Errorf("trace already running")
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create trace file: %w", err)
	}

	if err := trace.Start(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to start trace: %w", err)
	}

	t.file = f
	return nil
}

// Stop stops the trace and closes the file
func (t *Tracer) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.file == nil {
		return nil
	}

	trace.Stop()
	err := t.file.Close()
	t.file = nil
	return err
}

// IsRunning returns true if tracing is active
func (t *Tracer) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.file != nil
}
