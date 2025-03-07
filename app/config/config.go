package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	defaultTabWidth = 4
)

// GetTabWidth はタブ幅を取得する
func GetTabWidth() int {
	if width := os.Getenv("TAB_WIDTH"); width != "" {
		if val, err := strconv.Atoi(width); err == nil && val > 0 {
			return val
		}
	}
	return defaultTabWidth
}

// Config はエディタの設定を保持する構造体
type Config struct {
	TabWidth              int
	SmoothScroll          bool
	ScrollSteps           int
	DebugMode             bool
	StatusMessageDuration int // ステータスメッセージの表示時間（秒）
}

// LoadConfig は.envファイルから設定を読み込む
func LoadConfig() *Config {
	// .envファイルを読み込む
	godotenv.Load()

	config := &Config{
		TabWidth:              defaultTabWidth,
		SmoothScroll:          true,
		ScrollSteps:           3,
		DebugMode:             false,
		StatusMessageDuration: 5, // デフォルトは5秒
	}

	// TAB_WIDTHの環境変数を読み込む
	if tabWidth := os.Getenv("TAB_WIDTH"); tabWidth != "" {
		if width, err := strconv.Atoi(tabWidth); err == nil && width > 0 {
			config.TabWidth = width
		}
	}

	// SMOOTH_SCROLL環境変数から設定を読み込む
	if smooth := os.Getenv("SMOOTH_SCROLL"); smooth != "" {
		config.SmoothScroll = smooth != "0" && smooth != "false"
	}

	// SCROLL_STEPS環境変数から設定を読み込む
	if steps := os.Getenv("SCROLL_STEPS"); steps != "" {
		if val, err := strconv.Atoi(steps); err == nil && val > 0 {
			config.ScrollSteps = val
		}
	}

	// DEBUG環境変数から設定を読み込む
	if debug := os.Getenv("DEBUG"); debug != "" {
		config.DebugMode = debug == "true"
	}

	return config
}
