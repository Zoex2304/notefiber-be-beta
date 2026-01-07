package logger

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ILogger interface {
	Debug(module, message string, details map[string]interface{})
	Info(module, message string, details map[string]interface{})
	Warn(module, message string, details map[string]interface{})
	Error(module, message string, details map[string]interface{})
	Sync() error
	GetLogs(level string, limit, offset int) ([]LogEntry, error)
	GetLogById(id string) (*LogEntry, error)
}

type ZapLogger struct {
	logger   *zap.Logger
	filePath string
}

func NewZapLogger(logFilePath string, isProd bool) *ZapLogger {
	// 1. Configure Rotation (Lumberjack)
	rotator := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10,   // Megabytes
		MaxBackups: 5,    // Files
		MaxAge:     30,   // Days
		Compress:   true, // gzip
	}

	// 2. Configure Encoder (JSON)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.MessageKey = "message"
	encoderConfig.LevelKey = "level"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// 3. Configure Output Cores
	fileCore := zapcore.NewCore(
		jsonEncoder,
		zapcore.AddSync(rotator),
		zap.InfoLevel,
	)

	// Console Core
	var consoleEncoder zapcore.Encoder
	if isProd {
		consoleEncoder = jsonEncoder
	} else {
		consoleEncoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}

	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.Lock(os.Stdout),
		zap.DebugLevel,
	)

	// Join Cores (Tee)
	core := zapcore.NewTee(fileCore, consoleCore)

	// Create Logger
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)) // Skip 1 to point to caller of wrapper

	return &ZapLogger{
		logger:   l,
		filePath: logFilePath,
	}
}

// NewIsolatedLogger creates a logger that ONLY writes to the file, not console.
// This is useful for specific domain logs (e.g. WebSocket/Notification) to keep main logs clean.
func NewIsolatedLogger(logFilePath string) *ZapLogger {
	// 1. Configure Rotation (Lumberjack)
	rotator := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10,   // Megabytes
		MaxBackups: 5,    // Files
		MaxAge:     30,   // Days
		Compress:   true, // gzip
	}

	// 2. Configure Encoder (JSON)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.MessageKey = "message"
	encoderConfig.LevelKey = "level"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// 3. Configure Output Core (File Only)
	fileCore := zapcore.NewCore(
		jsonEncoder,
		zapcore.AddSync(rotator),
		zap.InfoLevel,
	)

	// Create Logger (No Console Core involved)
	l := zap.New(fileCore, zap.AddCaller(), zap.AddCallerSkip(1))

	return &ZapLogger{
		logger:   l,
		filePath: logFilePath,
	}
}

func (l *ZapLogger) Debug(module, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	l.logger.Debug(message, zap.String("module", module), zap.Any("details", details))
}

func (l *ZapLogger) Info(module, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	l.logger.Info(message, zap.String("module", module), zap.Any("details", details))
}

func (l *ZapLogger) Warn(module, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	l.logger.Warn(message, zap.String("module", module), zap.Any("details", details))
}

func (l *ZapLogger) Error(module, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	// Extract error from details if exists for stacktrace optimization
	if err, ok := details["error"]; ok {
		l.logger.Error(message, zap.String("module", module), zap.Any("details", details), zap.Any("error_ref", err))
	} else {
		l.logger.Error(message, zap.String("module", module), zap.Any("details", details))
	}
}

func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// Log Reading Capabilities for Admin Dashboard

type LogEntry struct {
	Id        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Module    string                 `json:"module,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func (l *ZapLogger) GetLogs(level string, limit, offset int) ([]LogEntry, error) {
	file, err := os.Open(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	// Read lines (This is naive for large files, but acceptable for MVP Admin)
	// For production, we should read reverse using a library or seek from end.
	// But JSON lines are variable length.
	// Let's read all, filter, and reverse in memory for now (assuming logs < 100MB active).
	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	// Buffer for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry LogEntry
		// Zap logs: {"level":"INFO","timestamp":"...","caller":"...","message":"...","module":"...","details":{...}}
		if err := json.Unmarshal(line, &entry); err == nil {
			// Filter by level if requested
			if level != "" && entry.Level != level {
				continue
			}
			// Add implicit ID if missing (hash of content?)
			if entry.Id == "" {
				entry.Id = fmt.Sprintf("%x", md5.Sum(line)) // basic ID
			}
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Reverse to show newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// Pagination
	start := offset
	end := offset + limit
	if start >= len(entries) {
		return []LogEntry{}, nil
	}
	if end > len(entries) {
		end = len(entries)
	}

	return entries[start:end], nil
}

func (l *ZapLogger) GetLogById(id string) (*LogEntry, error) {
	// Re-use logical scan (inefficient but works for now)
	logs, err := l.GetLogs("", 10000, 0) // Scan last 10k logs
	if err != nil {
		return nil, err
	}
	for _, log := range logs {
		if log.Id == id {
			return &log, nil
		}
	}
	return nil, fmt.Errorf("log not found")
}
