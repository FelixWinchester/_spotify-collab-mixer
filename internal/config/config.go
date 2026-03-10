package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config хранит все настройки приложения
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Port         string
}

// Load читает .env файл и возвращает конфиг
func Load() (*Config, error) {
	// Загружаем .env файл
	// Если файл не найден — не паникуем, возможно переменные уже в окружении
	_ = godotenv.Load()

	cfg := &Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("SPOTIFY_REDIRECT_URI"),
		Port:         os.Getenv("PORT"),
	}

	// Валидация — проверяем что все обязательные поля заполнены
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate проверяет что все обязательные поля присутствуют
func (c *Config) validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("SPOTIFY_CLIENT_ID is not set in .env file")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("SPOTIFY_CLIENT_SECRET is not set in .env file")
	}
	if c.RedirectURI == "" {
		return fmt.Errorf("SPOTIFY_REDIRECT_URI is not set in .env file")
	}
	if c.Port == "" {
		c.Port = "8888" // дефолтное значение если не указано
	}
	return nil
}
