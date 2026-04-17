package logger

import (
	"strings"
	"time"

	"go-seckill/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New 根据配置构造结构化日志实例。
// 这里统一输出 JSON，方便后续接日志平台、排查问题和做字段检索。
func New(cfg config.LogConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if err := level.Set(strings.ToLower(cfg.Level)); err != nil {
		return nil, err
	}

	zapConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: cfg.Development,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",
			LevelKey:   "level",
			TimeKey:    "time",
			NameKey:    "logger",
			CallerKey:  "caller",
			StacktraceKey: "stacktrace",
			EncodeLevel:   zapcore.LowercaseLevelEncoder,
			EncodeTime:    zapcore.ISO8601TimeEncoder,
			// 为了避免日志里出现微秒符号等非 ASCII 字符，这里统一记录为毫秒整数。
			EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendInt64(d.Milliseconds())
			},
			EncodeCaller:  zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return zapConfig.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
}
