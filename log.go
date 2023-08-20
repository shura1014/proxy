package proxy

import (
	"context"
	"github.com/shura1014/logger"
)

const (
	DebugLevel = logger.DebugLevel

	InfoLevel  = logger.InfoLevel
	WarnLevel  = logger.WarnLevel
	ErrorLevel = logger.ErrorLevel
	TEXT       = logger.TEXT
)

var (
	l   *logger.Logger
	ctx context.Context
)

func init() {
	l = logger.Default("proxy")
	ctx = context.TODO()
}

func Info(msg any, a ...any) {
	l.DoPrint(ctx, InfoLevel, msg, logger.GetFileNameAndLine(0), a...)
}

func Debug(msg any, a ...any) {
	l.DoPrint(ctx, DebugLevel, msg, logger.GetFileNameAndLine(0), a...)
}

func Error(msg any, a ...any) {
	l.DoPrint(ctx, ErrorLevel, msg, logger.GetFileNameAndLine(0), a...)
}

func Warn(msg any, a ...any) {
	l.DoPrint(ctx, WarnLevel, msg, logger.GetFileNameAndLine(0), a...)
}

func Text(msg any, a ...any) {
	l.DoPrint(ctx, TEXT, msg, logger.GetFileNameAndLine(0), a...)
}

func Fatal(msg any, a ...any) {
	l.DoPrint(ctx, ErrorLevel, msg, logger.GetFileNameAndLine(0), a...)
}
