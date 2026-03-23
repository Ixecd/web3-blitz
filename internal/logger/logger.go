package logger

import (
	"log/slog"
	"os"
)

// Init 根据环境变量初始化全局 slog。
//
//	LOG_LEVEL=debug  → 开启 Debug 级别（默认 Info）
//	LOG_FORMAT=text  → 本地开发友好的文本格式（默认 JSON）
func Init() {
	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
