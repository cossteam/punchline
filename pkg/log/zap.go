package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func SetupLogger(logLevel string) (*zap.Logger, error) {
	// 解析日志级别
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(logLevel)); err != nil {
		return nil, err
	}

	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zapLevel)

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "msg",
		LevelKey:     "level",
		TimeKey:      "time",
		CallerKey:    "caller",
		EncodeLevel:  zapcore.CapitalColorLevelEncoder,
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// 使用控制台作为输出
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleDebugging := zapcore.Lock(os.Stdout)

	core := zapcore.NewCore(consoleEncoder, consoleDebugging, atomicLevel)

	logger := zap.New(core, zap.AddCaller(), zap.Development())

	return logger, nil
}
