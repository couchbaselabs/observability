package configuration

import "go.uber.org/zap/zapcore"

// Config stores general configurable values for the manager
type Config struct {
	// Information about the store
	SQLiteKey string
	SQLiteDB  string
	// TLS certificate information
	CertPath string
	KeyPath  string
	// REST server information
	HTTPPort  int
	HTTPSPort int
	// Logging
	LogLevel zapcore.Level

	EncryptKey []byte
	SignKey    []byte
	UUID       string

	UIRoot string

	MaxWorkers int

	DisableHTTP  bool
	DisableHTTPS bool
}
