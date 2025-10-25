package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	// Start bot
	log.Println("✓ Bot initialized successfully")
	if err := b.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Bot error: %v", err)
	}

	log.Println("Bot stopped gracefully")
}
