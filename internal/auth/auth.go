package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelixWinchester/_spotify-collab-mixer/internal/config"
	"github.com/FelixWinchester/_spotify-collab-mixer/pkg/models"
)

const (
	spotifyAuthURL  = "https://accounts.spotify.com/authorize"
	spotifyTokenURL = "https://accounts.spotify.com/api/token"
	tokenFileName   = ".spotify_token.json"
)

// tokenWithExpiry хранит токен вместе со временем истечения
type tokenWithExpiry struct {
	models.SpotifyToken
	ExpiresAt time.Time `json:"expires_at"`
}

// Authenticator управляет OAuth flow
type Authenticator struct {
	cfg *config.Config
}

// New создаёт новый Authenticator
func New(cfg *config.Config) *Authenticator {
	return &Authenticator{cfg: cfg}
}

// GetValidToken возвращает валидный Access Token
func (a *Authenticator) GetValidToken() (string, error) {
	token, err := a.loadToken()
	if err == nil {
		if time.Now().Before(token.ExpiresAt) {
			fmt.Println("✅ Using existing token")
			return token.AccessToken, nil
		}
		fmt.Println("🔄 Token expired, refreshing...")
		newToken, err := a.refreshToken(token.RefreshToken)
		if err == nil {
			return newToken, nil
		}
		fmt.Printf("⚠️  Failed to refresh token: %v\n", err)
	}

	fmt.Println("🔐 Starting OAuth authorization flow...")
	return a.authorize()
}

// authorize запускает полный OAuth 2.0 Authorization Code Flow
func (a *Authenticator) authorize() (string, error) {
	state, err := generateRandomState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	authURL := a.buildAuthURL(state)
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + a.cfg.Port,
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errChan <- fmt.Errorf("state mismatch, possible CSRF attack")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errChan <- fmt.Errorf("spotify authorization error: %s", errParam)
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, successHTML)
		codeChan <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("\n🌐 Opening browser for Spotify authorization...\n")
	fmt.Printf("   If browser didn't open, go to:\n   %s\n\n", authURL)
	openBrowser(authURL)

	var authCode string
	select {
	case authCode = <-codeChan:
		fmt.Println("✅ Authorization code received!")
	case err := <-errChan:
		return "", fmt.Errorf("authorization failed: %w", err)
	case <-time.After(2 * time.Minute):
		return "", fmt.Errorf("authorization timeout: no response within 2 minutes")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	token, err := a.exchangeCode(authCode)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}

	if err := a.saveToken(token); err != nil {
		log.Printf("Warning: failed to save token: %v", err)
	}

	return token.AccessToken, nil
}

// buildAuthURL формирует URL для авторизации
func (a *Authenticator) buildAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", a.cfg.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", a.cfg.RedirectURI)
	params.Set("state", state)
	params.Set("scope", strings.Join(requiredScopes(), " "))
	return spotifyAuthURL + "?" + params.Encode()
}

// requiredScopes возвращает список необходимых разрешений
func requiredScopes() []string {
	return []string{
		"playlist-read-private",
		"playlist-read-collaborative",
		"playlist-modify-public",
		"playlist-modify-private",
	}
}

// exchangeCode обменивает authorization code на токены
func (a *Authenticator) exchangeCode(code string) (*tokenWithExpiry, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", a.cfg.RedirectURI)
	return a.requestToken(data)
}

// refreshToken обновляет Access Token через Refresh Token
func (a *Authenticator) refreshToken(refreshToken string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	token, err := a.requestToken(data)
	if err != nil {
		return "", err
	}

	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	if err := a.saveToken(token); err != nil {
		log.Printf("Warning: failed to save refreshed token: %v", err)
	}

	return token.AccessToken, nil
}

// requestToken выполняет HTTP запрос для получения токенов
func (a *Authenticator) requestToken(data url.Values) (*tokenWithExpiry, error) {
	req, err := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(a.cfg.ClientID, a.cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	var spotifyToken models.SpotifyToken
	if err := json.NewDecoder(resp.Body).Decode(&spotifyToken); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenWithExpiry{
		SpotifyToken: spotifyToken,
		ExpiresAt:    time.Now().Add(time.Duration(spotifyToken.ExpiresIn) * time.Second),
	}, nil
}

// saveToken сохраняет токен в файл
func (a *Authenticator) saveToken(token *tokenWithExpiry) error {
	tokenPath, err := getTokenPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	return os.WriteFile(tokenPath, data, 0600)
}

// loadToken загружает токен из файла
func (a *Authenticator) loadToken() (*tokenWithExpiry, error) {
	tokenPath, err := getTokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("token file not found: %w", err)
	}

	var token tokenWithExpiry
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &token, nil
}

// getTokenPath возвращает путь к файлу токена в домашней директории
func getTokenPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, tokenFileName), nil
}

// generateRandomState генерирует случайную строку для CSRF защиты
func generateRandomState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// openBrowser открывает URL в браузере на MacOS
func openBrowser(url string) {
	if err := exec.Command("open", url).Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

// successHTML — страница после успешной авторизации
const successHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Spotify Collab Mixer</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: #191414;
            color: white;
        }
        .container { text-align: center; padding: 40px; }
        .checkmark { font-size: 64px; }
        h1 { color: #1DB954; margin: 20px 0 10px; }
        p { color: #b3b3b3; }
    </style>
</head>
<body>
    <div class="container">
        <div class="checkmark">✅</div>
        <h1>Authorization Successful!</h1>
        <p>You can close this tab and return to the terminal.</p>
    </div>
</body>
</html>`
