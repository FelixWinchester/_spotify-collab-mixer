package main

import (
	"fmt"
	"log"
	"os"

	"github.com/FelixWinchester/_spotify-collab-mixer/internal/auth"
	"github.com/FelixWinchester/_spotify-collab-mixer/internal/config"
	"github.com/FelixWinchester/_spotify-collab-mixer/internal/spotify"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	fmt.Println("🎵 Spotify Collab Mixer")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")

	// Авторизация
	authenticator := auth.New(cfg)
	token, err := authenticator.GetValidToken()
	if err != nil {
		log.Fatalf("❌ Failed to authenticate: %v", err)
	}
	fmt.Println("✅ Authenticated!")

	// Создаём Spotify клиент
	client := spotify.New(token)

	// Получаем ID текущего пользователя
	userID, err := client.GetCurrentUserID()
	if err != nil {
		log.Fatalf("❌ Failed to get user ID: %v", err)
	}
	fmt.Printf("👤 Logged in as: %s\n", userID)

	// Тест — получаем плейлист по ID
	// Возьми любой ID плейлиста из своего Spotify
	// Открой плейлист в Spotify → Share → Copy link
	// Ссылка выглядит так: https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M
	// ID — это часть после /playlist/
	if len(os.Args) < 2 {
		fmt.Println("\n💡 Usage: go run cmd/mixer/main.go <playlist_id>")
		fmt.Println("   Example: go run cmd/mixer/main.go 37i9dQZF1DXcBWIGoYBM5M")
		os.Exit(0)
	}

	playlistID := os.Args[1]
	fmt.Printf("\n📋 Fetching playlist: %s\n", playlistID)

	playlist, err := client.GetPlaylist(playlistID)
	if err != nil {
		log.Fatalf("❌ Failed to get playlist: %v", err)
	}

	fmt.Printf("\n✅ Success!\n")
	fmt.Printf("   Playlist: %s\n", playlist.Name)
	fmt.Printf("   Tracks: %d\n", len(playlist.Tracks))

	if len(playlist.Tracks) > 0 {
		fmt.Printf("\n🎵 First 5 tracks:\n")
		for i, track := range playlist.Tracks {
			if i >= 5 {
				break
			}
			fmt.Printf("   %d. %s — %s\n", i+1, track.Name, track.Artists[0].Name)
		}
	}
}
