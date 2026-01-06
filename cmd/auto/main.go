package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/CastAIPhil/AUTO/internal/agent"
	"github.com/CastAIPhil/AUTO/internal/agent/providers/opencode"
	"github.com/CastAIPhil/AUTO/internal/alert"
	"github.com/CastAIPhil/AUTO/internal/config"
	"github.com/CastAIPhil/AUTO/internal/session"
	"github.com/CastAIPhil/AUTO/internal/store"
	"github.com/CastAIPhil/AUTO/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		configPath  string
		showVersion bool
	)

	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.StringVar(&configPath, "c", "", "Path to config file (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
	flag.Parse()

	if showVersion {
		fmt.Printf("AUTO version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if configPath == "" {
		configPath = config.ConfigPath()
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	st, err := store.New(cfg.Storage.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer st.Close()

	registry := agent.NewRegistry()

	if cfg.Providers.OpenCode.Enabled {
		provider := opencode.NewProvider(
			cfg.Providers.OpenCode.StoragePath,
			cfg.Providers.OpenCode.WatchInterval,
			cfg.Providers.OpenCode.MaxAge,
		)
		registry.Register(provider)
	}

	alertMgr := alert.NewManager(&cfg.Alerts, st)

	sessionMgr := session.NewManager(cfg, st, registry, alertMgr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sessionMgr.Start(ctx); err != nil {
		log.Fatalf("Failed to start session manager: %v", err)
	}

	app := tui.NewApp(cfg, sessionMgr, alertMgr)
	app.SetContext(ctx)

	sessionMgr.OnEvent(func(event agent.Event) {
		select {
		case app.EventChannel() <- event:
		default:
		}
	})

	alertMgr.OnAlert(func(a *alert.Alert) {
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	p := tea.NewProgram(app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
