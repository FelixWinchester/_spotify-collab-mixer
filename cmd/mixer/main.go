package main

import (
	"fmt"
	"log"

	"github.com/FelixWinchester/_spotify-collab-mixer/internal/config"
)

func main() {
	// Загружаем конфиг
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("✅ Config loaded successfully")
	fmt.Printf("   Client ID: %s...\n", cfg.ClientID[:8])
	fmt.Printf("   Redirect URI: %s\n", cfg.RedirectURI)
	fmt.Printf("   Port: %s\n", cfg.Port)
	fmt.Println("\n🎵 Spotify Collab Mixer is ready!")
}
