package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string, debug bool, logPath string) (*zap.Logger, error) {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}

	if debug {
		lvl = zapcore.DebugLevel
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core

	// File: always writes at configured level (JSON, no colors)
	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			fileEncoder := zapcore.NewJSONEncoder(encoderCfg)
			cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(file), zap.NewAtomicLevelAt(lvl)))
		}
	}

	// Console: error only (silent by default), colors in debug
	consoleLvl := zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	if debug {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleLvl = zap.NewAtomicLevelAt(lvl)
	}
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stderr), consoleLvl))

	core := zapcore.NewTee(cores...)
	return zap.New(core, zap.AddCaller()), nil
}
