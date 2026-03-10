package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/FelixWinchester/_spotify-collab-mixer/pkg/models"
)

const (
	baseURL             = "https://api.spotify.com/v1"
	maxTracksPerRequest = 100 // Spotify API лимит за один запрос
)

// Client — HTTP клиент для работы с Spotify API
type Client struct {
	accessToken string
	httpClient  *http.Client
}

// New создаёт новый Spotify клиент
func New(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- Структуры для парсинга ответов Spotify API ---

// apiPlaylist — ответ Spotify на запрос плейлиста
type apiPlaylist struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Items apiTracksPage `json:"items"`
}

// apiTracksPage — одна страница треков (Spotify возвращает по 100 штук)
type apiTracksPage struct {
	Items []apiTrackItem `json:"items"`
	Next  string         `json:"next"`
	Total int            `json:"total"`
}

// apiTrackItem — обёртка вокруг трека (Spotify оборачивает треки в объект)
type apiTrackItem struct {
	Item  apiTrack `json:"item"`
	Track apiTrack `json:"track"` // на случай если старый формат тоже встретится
}

// apiTrack — трек из Spotify API
type apiTrack struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	URI        string      `json:"uri"`
	Artists    []apiArtist `json:"artists"`
	Album      apiAlbum    `json:"album"`
	DurationMs int         `json:"duration_ms"`
}

// apiArtist — исполнитель из Spotify API
type apiArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// apiAlbum — альбом из Spotify API
type apiAlbum struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// apiUser — профиль пользователя
type apiUser struct {
	ID string `json:"id"`
}

// apiCreatePlaylistRequest — тело запроса для создания плейлиста
type apiCreatePlaylistRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
}

// apiAddTracksRequest — тело запроса для добавления треков
type apiAddTracksRequest struct {
	URIs []string `json:"uris"`
}

// apiCreatedPlaylist — ответ после создания плейлиста
type apiCreatedPlaylist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// --- Публичные методы клиента ---

// GetPlaylist возвращает плейлист со всеми треками по его ID
func (c *Client) GetPlaylist(playlistID string) (*models.Playlist, error) {
	// Получаем базовую информацию о плейлисте
	endpoint := fmt.Sprintf("%s/playlists/%s", baseURL, playlistID)

	var apiPl apiPlaylist
	if err := c.get(endpoint, &apiPl); err != nil {
		return nil, fmt.Errorf("failed to get playlist %s: %w", playlistID, err)
	}

	playlist := &models.Playlist{
		ID:   apiPl.ID,
		Name: apiPl.Name,
	}

	// Добавляем треки с первой страницы
	playlist.Tracks = append(playlist.Tracks, convertTracks(apiPl.Items.Items)...)

	// Если треков больше 100 — получаем остальные страницы
	nextURL := apiPl.Items.Next
	for nextURL != "" {
		var page apiTracksPage
		if err := c.get(nextURL, &page); err != nil {
			return nil, fmt.Errorf("failed to get tracks page: %w", err)
		}
		playlist.Tracks = append(playlist.Tracks, convertTracks(page.Items)...)
		nextURL = page.Next
	}

	fmt.Printf("   📋 \"%s\" — %d tracks\n", playlist.Name, len(playlist.Tracks))
	return playlist, nil
}

// GetCurrentUserID возвращает ID текущего пользователя
func (c *Client) GetCurrentUserID() (string, error) {
	var user apiUser
	if err := c.get(fmt.Sprintf("%s/me", baseURL), &user); err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return user.ID, nil
}

// CreatePlaylist создаёт новый плейлист для пользователя
func (c *Client) CreatePlaylist(userID, name, description string) (string, error) {
	endpoint := fmt.Sprintf("%s/users/%s/playlists", baseURL, userID)

	body := apiCreatePlaylistRequest{
		Name:        name,
		Description: description,
		Public:      false, // создаём приватным по умолчанию
	}

	var created apiCreatedPlaylist
	if err := c.post(endpoint, body, &created); err != nil {
		return "", fmt.Errorf("failed to create playlist: %w", err)
	}

	fmt.Printf("   ✅ Created playlist \"%s\" (ID: %s)\n", created.Name, created.ID)
	return created.ID, nil
}

// AddTracksToPlaylist добавляет треки в плейлист порциями по 100
func (c *Client) AddTracksToPlaylist(playlistID string, tracks []models.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	// Собираем все URI треков
	uris := make([]string, len(tracks))
	for i, track := range tracks {
		uris[i] = track.URI
	}

	// Разбиваем на порции по 100 (лимит Spotify API)
	for i := 0; i < len(uris); i += maxTracksPerRequest {
		end := i + maxTracksPerRequest
		if end > len(uris) {
			end = len(uris)
		}

		batch := uris[i:end]
		endpoint := fmt.Sprintf("%s/playlists/%s/tracks", baseURL, playlistID)

		if err := c.post(endpoint, apiAddTracksRequest{URIs: batch}, nil); err != nil {
			return fmt.Errorf("failed to add tracks batch %d-%d: %w", i, end, err)
		}

		fmt.Printf("   ➕ Added tracks %d-%d\n", i+1, end)
	}

	return nil
}

// --- Приватные HTTP методы ---

// get выполняет GET запрос и декодирует JSON ответ
func (c *Client) get(url string, result interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return err
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// post выполняет POST запрос с JSON телом
func (c *Client) post(url string, body interface{}, result interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return err
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// checkStatus проверяет HTTP статус ответа
func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Пробуем прочитать сообщение об ошибке от Spotify
	var apiErr struct {
		Error struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
		return fmt.Errorf("spotify API error %d: %s",
			apiErr.Error.Status,
			apiErr.Error.Message,
		)
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// --- Вспомогательные функции ---

// convertTracks конвертирует API треки в модели приложения
func convertTracks(items []apiTrackItem) []models.Track {
	tracks := make([]models.Track, 0, len(items))

	for _, item := range items {
		// Пропускаем пустые треки (бывает в Spotify API)
		// Поддерживаем оба формата: старый (track) и новый (item)
		t := item.Item
		if t.ID == "" {
			t = item.Track
		}
		if t.ID == "" {
			continue
		}

		artists := make([]models.Artist, len(t.Artists))
		for i, a := range t.Artists {
			artists[i] = models.Artist{
				ID:   a.ID,
				Name: a.Name,
			}
		}

		tracks = append(tracks, models.Track{
			ID:       t.ID,
			Name:     t.Name,
			URI:      t.URI,
			Artists:  artists,
			Album:    models.Album{ID: t.Album.ID, Name: t.Album.Name},
			Duration: t.DurationMs,
		})
	}

	return tracks
}
