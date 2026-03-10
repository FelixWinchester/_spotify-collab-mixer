# 🎵 Spotify Collab Mixer

A CLI tool written in Go that merges multiple Spotify playlists into one, removing duplicates intelligently.

## ✨ Features

- **Merge multiple playlists** — combine 2 or more playlists in a single command
- **Smart deduplication** — removes exact duplicates and near-duplicates (remastered versions, live editions, radio edits)
- **Pagination support** — handles playlists of any size
- **Token persistence** — OAuth token is saved locally, no need to re-authenticate every run
- **Automatic token refresh** — silently refreshes expired tokens

## 🛠 Tech Stack

- **Language:** Go 1.21+
- **Auth:** OAuth 2.0 Authorization Code Flow
- **API:** Spotify Web API
- **Dependencies:** `godotenv` for environment config

## 📋 Prerequisites

- Go 1.21 or higher
- A Spotify account
- A Spotify Developer app ([create one here](https://developer.spotify.com/dashboard))

## 🚀 Getting Started

### 1. Clone the repository
```bash
git clone https://github.com/FelixWinchester/_spotify-collab-mixer.git
cd _spotify-collab-mixer
```

### 2. Install dependencies
```bash
go mod download
```

### 3. Set up Spotify credentials

Create a `.env` file in the project root:
```bash
cp .env.example .env
```

Fill in your Spotify app credentials:
```env
SPOTIFY_CLIENT_ID=your_client_id_here
SPOTIFY_CLIENT_SECRET=your_client_secret_here
SPOTIFY_REDIRECT_URI=http://127.0.0.1:8888/callback
PORT=8888
```

> **Note:** In your Spotify Developer Dashboard, make sure `http://127.0.0.1:8888/callback` is added as a Redirect URI.

### 4. Build the binary
```bash
go build -o bin/mixer cmd/mixer/main.go
```

### 5. Run
```bash
./bin/mixer <playlist1_id> <playlist2_id> "New Playlist Name"
```

## 📖 Usage
```bash
# Merge two playlists
./bin/mixer ABC123 DEF456 "My Mixed Playlist"

# Merge three or more playlists
./bin/mixer ABC123 DEF456 GHI789 "Ultimate Mix"
```

### How to get a playlist ID

1. Open Spotify and navigate to any playlist
2. Click `•••` → **Share** → **Copy link to playlist**
3. The link looks like: `https://open.spotify.com/playlist/ABC123?si=xxx`
4. The ID is the part after `/playlist/` and before `?` → `ABC123`

## 📁 Project Structure
```
spotify-collab-mixer/
├── cmd/
│   └── mixer/
│       └── main.go          # Entry point
├── internal/
│   ├── auth/
│   │   └── auth.go          # OAuth 2.0 flow & token management
│   ├── spotify/
│   │   └── client.go        # Spotify API HTTP client
│   ├── playlist/
│   │   └── merger.go        # Merge & deduplication logic
│   └── config/
│       └── config.go        # Environment config loader
├── pkg/
│   └── models/
│       └── models.go        # Shared data models
├── .env.example             # Environment variables template
└── go.mod                   # Go module definition
```

## 🔐 Authentication

On first run, the app will open your browser to authenticate with Spotify. After you approve access, the token is saved to `~/.spotify_token.json` and reused on subsequent runs. The token is automatically refreshed when it expires.

## 🧠 How Deduplication Works

The merger uses two-pass deduplication:

1. **Exact match** — checks Spotify Track ID. Same track from different playlists is detected immediately.
2. **Fuzzy match** — normalizes track names by removing suffixes like `(Remastered 2011)`, `[Live]`, `- Radio Edit` and compares artist + normalized name. This catches the same song in different versions.

## 📄 License

MIT — see [LICENSE](LICENSE)
