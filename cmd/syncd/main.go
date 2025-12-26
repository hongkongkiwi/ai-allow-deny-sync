package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hongkongkiwi/codex-claude-allow-deny-sync/internal/config"
	"github.com/hongkongkiwi/codex-claude-allow-deny-sync/internal/sync"
)

func main() {
	var (
		configPath = flag.String("config", "syncd.yaml", "Path to config file")
		once       = flag.Bool("once", false, "Run one sync and exit")
		dryRun     = flag.Bool("dry-run", false, "Compute merged lists without writing changes")
		validate   = flag.Bool("validate", false, "Validate config and exit")
		interval   = flag.Duration("interval", 30*time.Second, "Sync interval")
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	if *validate {
		if err := sync.Validate(cfg); err != nil {
			log.Fatalf("validation error: %v", err)
		}
		fmt.Fprintln(os.Stdout, "config ok")
		return
	}

	if *once {
		policy, err := sync.Run(cfg, sync.Options{DryRun: *dryRun})
		if err != nil {
			log.Fatalf("sync error: %v", err)
		}
		if *dryRun {
			fmt.Fprintf(os.Stdout, "dry run complete (allow=%d, deny=%d)\n", len(policy.Allow), len(policy.Deny))
		} else {
			fmt.Fprintln(os.Stdout, "sync complete")
		}
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		if _, err := sync.Run(cfg, sync.Options{DryRun: *dryRun}); err != nil {
			log.Printf("sync error: %v", err)
		}
		<-ticker.C
	}
}
