package logger

import (
	log "github.com/sirupsen/logrus"
	"io"
)

type Level uint32

const (
	PanicLevel = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

var (
	logLevel  Level         = DebugLevel
	logFormat log.Formatter = &log.TextFormatter{}
)

type Logger struct {
	entry *log.Entry
}

func SetLevelAndFormat(l Level, formatter log.Formatter) {
	logLevel = l
	logFormat = formatter
}

func NewLogger(service string) *Logger {
	l := log.New()
	l.SetFormatter(logFormat)
	logger := &Logger{
		entry: l.WithField("service", service),
	}
	logger.SetLevel(logLevel)
	return logger
}

func (l *Logger) SetOutput(output io.Writer) {
	l.entry.Logger.SetOutput(output)
}

func (l *Logger) SetLevel(level Level) {
	l.entry.Logger.SetLevel(log.Level(level))
}

func (l *Logger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

func (l *Logger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.entry.Printf(format, args...)
}
