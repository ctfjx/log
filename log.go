// Package log is an extensible logging library
//
// We use the concept of "loggers sharing handlers" to enable
// more efficient use of resources for complex asynchronous
// logging needs.
//
// Usage:
//
//	import "github.com/LatteSec/log"
//
//	func main() {
//	  defer log.Sync()
//
//	  // Use it straight away with a default logger
//	  log.Info().Msg("Hello, World!").Send()
//	  log.Log(log.INFO).Msg("Hello, World!").Send()
//
//	  // or create a logger
//	  logger, _ := log.NewLogger().
//	              Name("my-logger").
//	              Level(log.INFO).
//	              Build()
//
//	  _ = logger.Start()
//	  logger.Info().Msg("Hello from custom logger!").Send()
//	  logger.Log(log.INFO).Msg("Hello from custom logger!").Send()
//
//	  // and you can register it to the global logger too!
//	  log.Register(logger)
//	}
//	```
package log

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
)

// Log Level
type Level int

// Log Levels
//
// Arranged from most to least verbose
const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	QUIET
)

var (
	levelNames = [6]string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "QUIET"}

	defaultLogger        atomic.Pointer[Logger]
	DefaultStdoutHandler atomic.Pointer[WriterHandler]
	DefaultStderrHandler atomic.Pointer[WriterHandler]

	handleInterrupts atomic.Bool

	ErrNotStarted                = errors.New("not started")
	ErrAlreadyStarted            = errors.New("already started")
	ErrInvalidLogHandler         = errors.New("invalid log handler")
	ErrInvalidLogLevel           = errors.New("invalid log level")
	ErrInvalidMaxFileSize        = errors.New("invalid max file size")
	ErrInvalidMaxFileArchives    = errors.New("invalid max file archives")
	ErrMissingLogFilename        = errors.New("missing log filename")
	ErrNoLogFileConfigured       = errors.New("no log file configured")
	ErrFoundDirWhenExpectingFile = errors.New("found directory when expecting file")
)

func init() {
	handleInterrupts.Store(true)
	go handleSigint()

	if err := RegisterStdoutHandler(NewWriterHandler(os.Stdout)); err != nil {
		panic(err)
	}
	if err := RegisterStderrHandler(NewWriterHandler(os.Stderr)); err != nil {
		panic(err)
	}
}

func DefaultLogger() *Logger {
	logger := defaultLogger.Load()
	if logger != nil {
		return logger
	}

	logger, err := NewLogger().Name("default").Build()
	if err != nil {
		panic(fmt.Errorf("could not build default logger: %v", err))
	}
	if err := logger.Start(); err != nil {
		panic(fmt.Errorf("could not start default logger: %v", err))
	}

	defaultLogger.Store(logger)
	logger.Info().Msg("default logger started").Send()

	return logger
}

// SetInterruptHandler enables or disables the SIGINT/TERM handler
//
// Please only disable this if you plan to implement your own handler.
// You can use [Sync] to handle cleanup on interrupt.
func SetInterruptHandler(enabled bool) {
	handleInterrupts.Store(enabled)
}

func Sync() {
	runCleanup()
}

func Register(l *Logger) {
	defaultLogger.Store(l)
}

func RegisterStdoutHandler(handler *WriterHandler) error {
	if err := handler.Start(); err != nil && err != ErrAlreadyStarted {
		return err
	}
	DefaultStdoutHandler.Store(handler)
	return nil
}

func RegisterStderrHandler(handler *WriterHandler) error {
	if err := handler.Start(); err != nil && err != ErrAlreadyStarted {
		return err
	}
	DefaultStderrHandler.Store(handler)
	return nil
}

func Log(level Level) *LogMessage { return DefaultLogger().Log(level) }
func Debug() *LogMessage          { return DefaultLogger().Debug() }
func Info() *LogMessage           { return DefaultLogger().Info() }
func Warn() *LogMessage           { return DefaultLogger().Warn() }
func Error() *LogMessage          { return DefaultLogger().Error() }
func Fatal() *LogMessage          { return DefaultLogger().Fatal() }
