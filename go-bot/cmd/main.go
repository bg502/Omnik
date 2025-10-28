package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/drew/omnik-bot/internal/api"
	"github.com/drew/omnik-bot/internal/bot"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting omnik Go bot...")

	// Load configuration
	cfg, err := bot.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create bot
	b, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping bot...")
		cancel()
	}()

	// Check if HTTP API is enabled
	apiPort := 0
	if portStr := os.Getenv("OMNI_API_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			apiPort = p
		}
	}

	// Start HTTP API server if enabled
	if apiPort > 0 {
		apiServer := api.New(apiPort, func(ctx context.Context, message string, sessionID string) error {
			return b.ProcessAPIMessage(ctx, message, sessionID)
		})

		go func() {
			log.Printf("Starting HTTP API server on port %d", apiPort)
			if err := apiServer.Start(ctx); err != nil {
				log.Printf("API server error: %v", err)
			}
		}()
	}

	// Start bot
	log.Println("✓ Bot initialized successfully")
	if err := b.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Bot error: %v", err)
	}

	log.Println("Bot stopped gracefully")
}
