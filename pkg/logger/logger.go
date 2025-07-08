package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// NewLogger creates a new configured logger instance
func NewLogger(level, format string) *logrus.Logger {
	log := logrus.New()

	// Set output to stdout
	log.SetOutput(os.Stdout)

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)

	// Set log format
	switch format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	default:
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	return log
}

// WithComponent adds a component field to log entries
func WithComponent(log *logrus.Logger, component string) *logrus.Entry {
	return log.WithField("component", component)
}

// WithError adds an error field to log entries
func WithError(log *logrus.Logger, err error) *logrus.Entry {
	return log.WithError(err)
}

// WithFields adds multiple fields to log entries
func WithFields(log *logrus.Logger, fields logrus.Fields) *logrus.Entry {
	return log.WithFields(fields)
}
