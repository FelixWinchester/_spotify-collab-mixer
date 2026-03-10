package models

// Track представляет один трек из Spotify
type Track struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Artists  []Artist `json:"artists"`
	Album    Album    `json:"album"`
	Duration int      `json:"duration_ms"`
	URI      string   `json:"uri"`
}

// Artist представляет исполнителя
type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Album представляет альбом
type Album struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Playlist представляет плейлист Spotify
type Playlist struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Tracks []Track `json:"tracks"`
}

// SpotifyToken хранит OAuth токены
type SpotifyToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}
