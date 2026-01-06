package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type StreamEvent struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionID,omitempty"`
	MessageID string          `json:"messageID,omitempty"`
	PartID    string          `json:"partID,omitempty"`
	Text      string          `json:"text,omitempty"`
	ToolName  string          `json:"toolName,omitempty"`
	State     string          `json:"state,omitempty"`
	Error     string          `json:"error,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

type RunConfig struct {
	SessionID string
	Directory string
	Message   string
	Model     string
	Agent     string
	Title     string
}

type Runner struct {
	cmd       *exec.Cmd
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	events    chan StreamEvent
	errors    chan error
	done      chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	running   bool
	sessionID string
}

func NewRunner(ctx context.Context, cfg RunConfig) (*Runner, error) {
	ctx, cancel := context.WithCancel(ctx)

	args := []string{"run", "--format", "json"}

	if cfg.SessionID != "" {
		args = append(args, "-s", cfg.SessionID)
	}

	if cfg.Model != "" {
		args = append(args, "-m", cfg.Model)
	}

	if cfg.Agent != "" {
		args = append(args, "--agent", cfg.Agent)
	}

	if cfg.Title != "" {
		args = append(args, "--title", cfg.Title)
	}

	if cfg.Message != "" {
		args = append(args, cfg.Message)
	}

	cmd := exec.CommandContext(ctx, "opencode", args...)
	if cfg.Directory != "" {
		cmd.Dir = cfg.Directory
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	return &Runner{
		cmd:       cmd,
		stdout:    stdout,
		stderr:    stderr,
		events:    make(chan StreamEvent, 100),
		errors:    make(chan error, 10),
		done:      make(chan struct{}),
		ctx:       ctx,
		cancel:    cancel,
		sessionID: cfg.SessionID,
	}, nil
}

func (r *Runner) Start() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("runner already started")
	}
	r.running = true
	r.mu.Unlock()

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	go r.streamOutput()
	go r.streamErrors()
	go r.waitForCompletion()

	return nil
}

func (r *Runner) streamOutput() {
	scanner := bufio.NewScanner(r.stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event StreamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			event = StreamEvent{
				Type: "raw",
				Text: string(line),
			}
		}
		event.Raw = json.RawMessage(line)

		if event.SessionID != "" && r.sessionID == "" {
			r.mu.Lock()
			r.sessionID = event.SessionID
			r.mu.Unlock()
		}

		select {
		case r.events <- event:
		case <-r.ctx.Done():
			return
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		select {
		case r.errors <- fmt.Errorf("stdout scan error: %w", err):
		default:
		}
	}
}

func (r *Runner) streamErrors() {
	scanner := bufio.NewScanner(r.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			select {
			case r.events <- StreamEvent{Type: "stderr", Text: line}:
			case <-r.ctx.Done():
				return
			}
		}
	}
}

func (r *Runner) waitForCompletion() {
	err := r.cmd.Wait()
	if err != nil {
		select {
		case r.errors <- err:
		default:
		}
	}

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()

	close(r.done)
}

func (r *Runner) Events() <-chan StreamEvent {
	return r.events
}

func (r *Runner) Errors() <-chan error {
	return r.errors
}

func (r *Runner) Done() <-chan struct{} {
	return r.done
}

func (r *Runner) Stop() {
	r.cancel()
}

func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

func (r *Runner) SessionID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessionID
}

func (r *Runner) Wait() error {
	<-r.done
	select {
	case err := <-r.errors:
		return err
	default:
		return nil
	}
}
