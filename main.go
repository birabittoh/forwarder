package main

import (
	"log"

	"github.com/birabittoh/forwarder/config"
	"github.com/birabittoh/forwarder/forwarder"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	log.Println("Starting Telegram Forwarder Bot...")

	// Load configuration
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Configuration loaded:")
	log.Printf("- Source Channel: %d", cfg.SourceChannelID)
	log.Printf("- Target Channel: %d", cfg.TargetChannelID)
	log.Printf("- Discussion Group: %d", cfg.DiscussionGroupID)
	log.Printf("- Ignore Regex: %s", cfg.IgnoreRegex)

	// Initialize TDLib
	forwarder, err := forwarder.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize TDLib: %v", err)
	}

	log.Println("TDLib initialized successfully")

	forwarder.Listen()
}
