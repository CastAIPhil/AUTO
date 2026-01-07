// Package debug provides runtime profiling and diagnostics for AUTO
package debug

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

// Profiler manages the pprof HTTP server for runtime profiling
type Profiler struct {
	addr   string
	server *http.Server
	mu     sync.Mutex
}

// NewProfiler creates a new profiler that will listen on the given address
// Default address is "localhost:6060" if empty string is passed
func NewProfiler(addr string) *Profiler {
	if addr == "" {
		addr = "localhost:6060"
	}
	return &Profiler{
		addr: addr,
	}
}

// Start starts the pprof HTTP server in a background goroutine
// The server exposes the following endpoints:
//   - /debug/pprof/              - Index page with links to all profiles
//   - /debug/pprof/profile       - CPU profile (add ?seconds=N for duration)
//   - /debug/pprof/heap          - Heap memory profile
//   - /debug/pprof/goroutine     - Goroutine stack traces
//   - /debug/pprof/block         - Block profile (requires runtime.SetBlockProfileRate)
//   - /debug/pprof/mutex         - Mutex contention profile (requires runtime.SetMutexProfileFraction)
//   - /debug/pprof/threadcreate  - Thread creation profile
//   - /debug/pprof/trace         - Execution trace (add ?seconds=N for duration)
//
// Usage from another terminal:
//
//	go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
//	go tool pprof http://localhost:6060/debug/pprof/heap
//	curl http://localhost:6060/debug/pprof/goroutine?debug=2
func (p *Profiler) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.server != nil {
		return fmt.Errorf("profiler already running")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)

	p.server = &http.Server{
		Addr:         p.addr,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("[profiler] Starting pprof server on http://%s/debug/pprof/", p.addr)
		log.Printf("[profiler] CPU profile: go tool pprof http://%s/debug/pprof/profile?seconds=30", p.addr)
		log.Printf("[profiler] Heap profile: go tool pprof http://%s/debug/pprof/heap", p.addr)
		log.Printf("[profiler] Goroutines:   curl http://%s/debug/pprof/goroutine?debug=2", p.addr)

		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[profiler] Server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the pprof server
func (p *Profiler) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.server == nil {
		return nil
	}

	log.Printf("[profiler] Shutting down pprof server")
	err := p.server.Shutdown(ctx)
	p.server = nil
	return err
}

// Addr returns the address the profiler is listening on
func (p *Profiler) Addr() string {
	return p.addr
}

// IsRunning returns true if the profiler server is running
func (p *Profiler) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.server != nil
}
