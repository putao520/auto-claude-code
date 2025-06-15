package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志器接口
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	With(fields ...zap.Field) Logger
	Sync() error
	GetZapLogger() *zap.Logger
}

// zapLogger zap 日志器包装
type zapLogger struct {
	logger *zap.Logger
}

// NewLogger 创建新的日志器
func NewLogger(level string, debug bool) (Logger, error) {
	var config zap.Config

	if debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// 设置日志级别
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		config.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// 构建日志器
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &zapLogger{logger: logger}, nil
}

// NewConsoleLogger 创建控制台日志器
func NewConsoleLogger(level string) (Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 设置日志级别
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		config.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &zapLogger{logger: logger}, nil
}

// NewFileLogger 创建文件日志器
func NewFileLogger(level, filePath string) (Logger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{filePath}
	config.ErrorOutputPaths = []string{filePath}
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 设置日志级别
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		config.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &zapLogger{logger: logger}, nil
}

// Debug 记录调试日志
func (l *zapLogger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

// Info 记录信息日志
func (l *zapLogger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

// Warn 记录警告日志
func (l *zapLogger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

// Error 记录错误日志
func (l *zapLogger) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

// Fatal 记录致命错误日志并退出程序
func (l *zapLogger) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

// With 添加字段到日志器
func (l *zapLogger) With(fields ...zap.Field) Logger {
	return &zapLogger{logger: l.logger.With(fields...)}
}

// Sync 同步日志缓冲区
func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

// GetZapLogger 获取底层的 zap.Logger（用于需要直接使用 zap 的场景）
func (l *zapLogger) GetZapLogger() *zap.Logger {
	return l.logger
}

// 全局日志器实例
var globalLogger Logger

// InitGlobalLogger 初始化全局日志器
func InitGlobalLogger(level string, debug bool) error {
	logger, err := NewLogger(level, debug)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger 获取全局日志器
func GetGlobalLogger() Logger {
	if globalLogger == nil {
		// 如果全局日志器未初始化，创建一个默认的控制台日志器
		logger, err := NewConsoleLogger("info")
		if err != nil {
			// 如果创建失败，使用 zap 的默认日志器
			globalLogger = &zapLogger{logger: zap.NewNop()}
		} else {
			globalLogger = logger
		}
	}
	return globalLogger
}

// Debug 使用全局日志器记录调试日志
func Debug(msg string, fields ...zap.Field) {
	GetGlobalLogger().Debug(msg, fields...)
}

// Info 使用全局日志器记录信息日志
func Info(msg string, fields ...zap.Field) {
	GetGlobalLogger().Info(msg, fields...)
}

// Warn 使用全局日志器记录警告日志
func Warn(msg string, fields ...zap.Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

// Error 使用全局日志器记录错误日志
func Error(msg string, fields ...zap.Field) {
	GetGlobalLogger().Error(msg, fields...)
}

// Fatal 使用全局日志器记录致命错误日志并退出程序
func Fatal(msg string, fields ...zap.Field) {
	GetGlobalLogger().Fatal(msg, fields...)
}

// Sync 同步全局日志器缓冲区
func Sync() error {
	return GetGlobalLogger().Sync()
}

// SetGlobalLogger 设置全局日志器
func SetGlobalLogger(logger Logger) {
	globalLogger = logger
}

// CreateLoggerFromConfig 从配置创建日志器
func CreateLoggerFromConfig(level string, debug bool, logFile string) (Logger, error) {
	if logFile != "" {
		return NewFileLogger(level, logFile)
	}

	if debug {
		return NewConsoleLogger(level)
	}

	return NewLogger(level, debug)
}

// LoggerMiddleware 日志中间件（为后续 HTTP 服务器使用）
func LoggerMiddleware(logger Logger) func(next func()) func() {
	return func(next func()) func() {
		return func() {
			// 在这里可以添加请求日志记录逻辑
			next()
		}
	}
}

// WithError 添加错误字段的便捷方法
func WithError(err error) zap.Field {
	return zap.Error(err)
}

// WithString 添加字符串字段的便捷方法
func WithString(key, value string) zap.Field {
	return zap.String(key, value)
}

// WithInt 添加整数字段的便捷方法
func WithInt(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// WithBool 添加布尔字段的便捷方法
func WithBool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

// WithDuration 添加时间间隔字段的便捷方法
func WithDuration(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}
