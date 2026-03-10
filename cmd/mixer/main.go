package main

import (
	"fmt"
	"log"

	"github.com/FelixWinchester/_spotify-collab-mixer/internal/auth"
	"github.com/FelixWinchester/_spotify-collab-mixer/internal/config"
)

func main() {
	// Загружаем конфиг
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	fmt.Println("🎵 Spotify Collab Mixer")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")

	// Инициализируем авторизацию
	authenticator := auth.New(cfg)

	// Получаем валидный токен
	token, err := authenticator.GetValidToken()
	if err != nil {
		log.Fatalf("❌ Failed to authenticate: %v", err)
	}

	fmt.Printf("✅ Authenticated successfully!\n")
	fmt.Printf("   Token: %s...\n", token[:20])
	fmt.Println("\n🚀 Ready to mix playlists!")
}
