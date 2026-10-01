package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func NewLogger(env string) *slog.Logger {
	var handler slog.Handler
	if strings.ToLower(env) == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	fmt.Printf("handler %v\n", handler)
	log := slog.New(handler)
	slog.SetDefault(log)
	return log
}
