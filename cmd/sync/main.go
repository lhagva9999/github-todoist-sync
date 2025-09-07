package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github-todoist-sync/internal/config"
	"github-todoist-sync/internal/sync"
)

func main() {
	var (
		mode    = flag.String("mode", "once", "Režim spuštění: 'once', 'daemon', 'github-only', 'todoist-only'")
		verbose = flag.Bool("verbose", false, "Podrobné logování")
	)
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Načteme konfiguraci
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Chyba při načítání konfigurace: %v", err)
	}

	if cfg.App.Debug {
		log.Printf("Debug režim zapnut")
		log.Printf("GitHub: %s/%s", cfg.GitHub.Owner, cfg.GitHub.Repo)
		log.Printf("Todoist projekt: %s", cfg.Todoist.ProjectName)
		log.Printf("Interval synchronizace: %v", cfg.App.SyncInterval)
	}

	// Vytvoříme synchronizační službu
	syncService, err := sync.NewService(cfg)
	if err != nil {
		log.Fatalf("Chyba při inicializaci služby: %v", err)
	}

	ctx := context.Background()

	switch *mode {
	case "once":
		log.Printf("Spouštím jednorazovou synchronizaci...")
		if err := syncService.FullSync(ctx); err != nil {
			log.Fatalf("Chyba při synchronizaci: %v", err)
		}
		log.Printf("Synchronizace dokončena")

	case "github-only":
		log.Printf("Spouštím synchronizaci pouze GitHub → Todoist...")
		if err := syncService.SyncFromGitHub(ctx); err != nil {
			log.Fatalf("Chyba při synchronizaci z GitHub: %v", err)
		}
		log.Printf("Synchronizace z GitHub dokončena")

	case "todoist-only":
		log.Printf("Spouštím synchronizaci pouze Todoist → GitHub...")
		if err := syncService.SyncToGitHub(ctx); err != nil {
			log.Fatalf("Chyba při synchronizaci do GitHub: %v", err)
		}
		log.Printf("Synchronizace do GitHub dokončena")

	case "daemon":
		log.Printf("Spouštím službu v daemon režimu (interval: %v)...", cfg.App.SyncInterval)
		runDaemon(ctx, syncService, cfg.App.SyncInterval)

	default:
		fmt.Fprintf(os.Stderr, "Neplatný režim: %s\nPovolené režimy: once, daemon, github-only, todoist-only\n", *mode)
		os.Exit(1)
	}
}

func runDaemon(ctx context.Context, syncService *sync.Service, interval time.Duration) {
	// Nastavíme zachytávání signálů pro graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Vytvoříme ticker pro pravidelnou synchronizaci
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Daemon spuštěn. Pro ukončení stiskněte Ctrl+C")

	// Provedeme první synchronizaci ihned
	log.Printf("Provádím počáteční synchronizaci...")
	if err := syncService.FullSync(ctx); err != nil {
		log.Printf("Chyba při počáteční synchronizaci: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			log.Printf("Spouštím plánovanou synchronizaci...")
			if err := syncService.FullSync(ctx); err != nil {
				log.Printf("Chyba při synchronizaci: %v", err)
			}

		case sig := <-sigChan:
			log.Printf("Přijat signál %v, ukončuji...", sig)
			return
		}
	}
}
