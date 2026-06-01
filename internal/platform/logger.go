package platform

import (
	"os"

	"github.com/natefinch/lumberjack.v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(env string) (*zap.Logger, error) {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if env == "development" {
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/sola.log",
		MaxSize:    20,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	})

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), writer),
		level,
	)
	return zap.New(core), nil
}
