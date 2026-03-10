package main

import (
	"fmt"
	"log"
	"os"

	"github.com/FelixWinchester/_spotify-collab-mixer/internal/auth"
	"github.com/FelixWinchester/_spotify-collab-mixer/internal/config"
	"github.com/FelixWinchester/_spotify-collab-mixer/internal/playlist"
	"github.com/FelixWinchester/_spotify-collab-mixer/internal/spotify"
	"github.com/FelixWinchester/_spotify-collab-mixer/pkg/models"
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

	// Проверяем аргументы
	// Минимум 3 аргумента: playlist1 playlist2 "New Playlist Name"
	if len(os.Args) < 4 {
		fmt.Println("\n💡 Usage:")
		fmt.Println("   go run cmd/mixer/main.go <playlist1_id> <playlist2_id> ... <new_playlist_name>")
		fmt.Println("\n   Example:")
		fmt.Println("   go run cmd/mixer/main.go ABC123 DEF456 \"My Mixed Playlist\"")
		fmt.Println("\n   You can merge 2 or more playlists at once.")
		os.Exit(0)
	}

	// Последний аргумент — название нового плейлиста
	// Все предыдущие — ID плейлистов
	playlistIDs := os.Args[1 : len(os.Args)-1]
	newPlaylistName := os.Args[len(os.Args)-1]

	// Создаём Spotify клиент
	client := spotify.New(token)

	// Получаем ID текущего пользователя
	userID, err := client.GetCurrentUserID()
	if err != nil {
		log.Fatalf("❌ Failed to get user ID: %v", err)
	}
	fmt.Printf("👤 User: %s\n", userID)

	// Загружаем все плейлисты
	fmt.Printf("\n📥 Fetching %d playlists...\n", len(playlistIDs))
	playlists := make([]*models.Playlist, 0, len(playlistIDs))

	for _, id := range playlistIDs {
		pl, err := client.GetPlaylist(id)
		if err != nil {
			log.Fatalf("❌ Failed to get playlist %s: %v", id, err)
		}
		playlists = append(playlists, pl)
	}

	// Запускаем слияние
	fmt.Println("\n🔀 Merging playlists...")
	merger := playlist.New(false) // false = умная дедупликация
	result := merger.Merge(playlists)

	// Выводим статистику
	result.PrintStats()

	// Создаём новый плейлист в Spotify
	fmt.Printf("\n📤 Creating new playlist \"%s\"...\n", newPlaylistName)

	description := fmt.Sprintf(
		"Mixed from %d playlists by Spotify Collab Mixer. %d unique tracks.",
		len(playlists),
		result.UniqueCount,
	)

	newPlaylistID, err := client.CreatePlaylist(userID, newPlaylistName, description)
	if err != nil {
		log.Fatalf("❌ Failed to create playlist: %v", err)
	}

	// Добавляем треки в новый плейлист
	fmt.Printf("➕ Adding %d tracks to playlist...\n", result.UniqueCount)
	if err := client.AddTracksToPlaylist(newPlaylistID, result.Tracks); err != nil {
		log.Fatalf("❌ Failed to add tracks: %v", err)
	}

	// Финальное сообщение
	fmt.Printf("\n✅ Done! Playlist \"%s\" created successfully!\n", newPlaylistName)
	fmt.Printf("🎵 Open in Spotify: https://open.spotify.com/playlist/%s\n", newPlaylistID)
}
