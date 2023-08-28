package slogrotate

import (
	"log/slog"

	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLumberjackLogger(
	filepath string,
	maxBackups, maxAge, maxSize int,
	localtime, compress bool,
) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filepath,
		LocalTime:  localtime,
		Compress:   compress,
		MaxSize:    maxSize,
		MaxAge:     maxAge,
		MaxBackups: maxBackups,
	}
}

func NewSlogRotateLogger(lumberjackLogger *lumberjack.Logger) *slog.Logger {
	return slog.New(slog.NewTextHandler(lumberjackLogger, &slog.HandlerOptions{}))
}
