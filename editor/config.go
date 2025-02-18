package editor

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config はエディタの設定を保持する構造体
type Config struct {
	TabWidth int
}

// LoadConfig は.envファイルから設定を読み込む
func LoadConfig() *Config {
	// .envファイルを読み込む
	godotenv.Load()

	config := &Config{
		TabWidth: 2, // デフォルト値
	}

	// TAB_WIDTHの環境変数を読み込む
	if tabWidth := os.Getenv("TAB_WIDTH"); tabWidth != "" {
		if width, err := strconv.Atoi(tabWidth); err == nil && width > 0 {
			config.TabWidth = width
		}
	}

	return config
}
