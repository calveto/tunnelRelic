package tunnelRelic

import (
	"log"
	"os"
)

type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	EnableDebug()
}

type StderrLogger struct {
	log        *log.Logger
	DebugLevel bool
}

func NewStderrLogger() *StderrLogger {
	l := new(StderrLogger)
	l.log = log.New(os.Stderr, "tunnelRelic: ", 0)
	return l
}

func (l *StderrLogger) EnableDebug() {
	l.DebugLevel = true
}

func (l *StderrLogger) Info(args ...interface{}) {
	l.log.Print(append([]interface{}{"[INFO] "}, args...)...)
}

func (l *StderrLogger) Infof(format string, args ...interface{}) {
	l.log.Printf("[INFO] "+format, args...)
}

func (l *StderrLogger) Debug(args ...interface{}) {
	if l.DebugLevel {
		l.log.Print(append([]interface{}{"[DEBUG] "}, args...)...)
	}
}

func (l *StderrLogger) Debugf(format string, args ...interface{}) {
	if l.DebugLevel {
		l.log.Printf("[DEBUG] "+format, args...)
	}
}

func (l *StderrLogger) Warn(args ...interface{}) {
	l.log.Print(append([]interface{}{"[WARN] "}, args...)...)
}

func (l *StderrLogger) Warnf(format string, args ...interface{}) {
	l.log.Printf("[WARN] "+format, args...)
}

func (l *StderrLogger) Error(args ...interface{}) {
	l.log.Print(append([]interface{}{"[ERROR] "}, args...)...)
}

func (l *StderrLogger) Errorf(format string, args ...interface{}) {
	l.log.Printf("[ERROR] "+format, args...)
}
