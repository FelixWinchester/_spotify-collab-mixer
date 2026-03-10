package playlist

import (
	"fmt"
	"strings"

	"github.com/FelixWinchester/_spotify-collab-mixer/pkg/models"
)

// MergeResult содержит результат слияния и статистику
type MergeResult struct {
	Tracks          []models.Track
	TotalInput      int          // сколько треков было на входе (с дублями)
	DuplicatesFound int          // сколько дублей убрали
	UniqueCount     int          // сколько уникальных треков в итоге
	SourceStats     []SourceStat // статистика по каждому плейлисту
}

// SourceStat статистика по одному исходному плейлисту
type SourceStat struct {
	PlaylistName string
	TrackCount   int
}

// Merger объединяет плейлисты
type Merger struct {
	// StrictMode = true  — дедупликация только по Track ID (100% совпадение)
	// StrictMode = false — дедупликация по ID + по нормализованному имени (ловит ремастеры и тд)
	StrictMode bool
}

// New создаёт новый Merger
// strictMode = false по умолчанию — умная дедупликация
func New(strictMode bool) *Merger {
	return &Merger{StrictMode: strictMode}
}

// Merge объединяет несколько плейлистов в один без дубликатов
func (m *Merger) Merge(playlists []*models.Playlist) *MergeResult {
	result := &MergeResult{}

	// Собираем статистику по источникам
	for _, pl := range playlists {
		result.SourceStats = append(result.SourceStats, SourceStat{
			PlaylistName: pl.Name,
			TrackCount:   len(pl.Tracks),
		})
		result.TotalInput += len(pl.Tracks)
	}

	// Два индекса для дедупликации:
	// seenIDs — для точного совпадения по Spotify Track ID
	// seenNormalized — для нечёткого совпадения по нормализованному имени
	seenIDs := make(map[string]bool)
	seenNormalized := make(map[string]bool)

	uniqueTracks := make([]models.Track, 0)

	for _, pl := range playlists {
		for _, track := range pl.Tracks {
			// Шаг 1 — проверяем точное совпадение по ID
			if seenIDs[track.ID] {
				result.DuplicatesFound++
				continue
			}

			// Шаг 2 — если не строгий режим, проверяем нормализованное имя
			if !m.StrictMode {
				normalizedKey := normalizeTrack(track)
				if seenNormalized[normalizedKey] {
					result.DuplicatesFound++
					continue
				}
				seenNormalized[normalizedKey] = true
			}

			// Трек уникален — добавляем
			seenIDs[track.ID] = true
			uniqueTracks = append(uniqueTracks, track)
		}
	}

	result.Tracks = uniqueTracks
	result.UniqueCount = len(uniqueTracks)

	return result
}

// normalizeTrack создаёт нормализованный ключ для трека
// Это позволяет поймать дубли вида:
// "Bohemian Rhapsody" и "Bohemian Rhapsody - Remastered 2011"
// "Lose Yourself" и "Lose Yourself (From 8 Mile)"
func normalizeTrack(track models.Track) string {
	name := normalizeName(track.Name)

	// Берём первого исполнителя для ключа
	artist := ""
	if len(track.Artists) > 0 {
		artist = strings.ToLower(strings.TrimSpace(track.Artists[0].Name))
	}

	return fmt.Sprintf("%s|%s", artist, name)
}

// normalizeName убирает из названия трека всё лишнее
func normalizeName(name string) string {
	name = strings.ToLower(name)

	// Убираем скобки и всё что внутри них
	// "Song (Remastered 2011)" -> "Song"
	// "Song [Live]" -> "Song"
	for {
		start := strings.IndexByte(name, '(')
		end := strings.IndexByte(name, ')')
		if start != -1 && end != -1 && end > start {
			name = name[:start] + name[end+1:]
		} else {
			break
		}
	}

	for {
		start := strings.IndexByte(name, '[')
		end := strings.IndexByte(name, ']')
		if start != -1 && end != -1 && end > start {
			name = name[:start] + name[end+1:]
		} else {
			break
		}
	}

	// Убираем типичные суффиксы ремастеров через дефис
	// "Song - Remastered" -> "Song"
	suffixes := []string{
		" - remastered",
		" - radio edit",
		" - live",
		" - acoustic",
		" - remix",
		" - edit",
		" - version",
		" - original",
		" - single",
		" - bonus track",
	}

	for _, suffix := range suffixes {
		if idx := strings.Index(name, suffix); idx != -1 {
			name = name[:idx]
		}
	}

	// Убираем лишние пробелы
	name = strings.Join(strings.Fields(name), " ")
	name = strings.TrimSpace(name)

	return name
}

// PrintStats выводит статистику слияния в терминал
func (r *MergeResult) PrintStats() {
	fmt.Println("\n📊 Merge Statistics:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n📋 Source playlists:")
	for _, stat := range r.SourceStats {
		fmt.Printf("   • %-30s %d tracks\n", stat.PlaylistName, stat.TrackCount)
	}

	fmt.Println("\n📈 Results:")
	fmt.Printf("   Total input tracks:   %d\n", r.TotalInput)
	fmt.Printf("   Duplicates removed:   %d\n", r.DuplicatesFound)
	fmt.Printf("   Unique tracks:        %d\n", r.UniqueCount)

	if r.TotalInput > 0 {
		percentage := float64(r.DuplicatesFound) / float64(r.TotalInput) * 100
		fmt.Printf("   Duplicate rate:       %.1f%%\n", percentage)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
