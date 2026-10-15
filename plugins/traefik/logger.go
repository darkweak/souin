package traefik

import (
	"fmt"
	"os"
	"strings"

	"github.com/darkweak/storages/core"
)

const (
	debugLevel = iota
	infoLevel
	warnLevel
	errorLevel
)

// logger is a minimal core.Logger, zap can't be interpreted by Yaegi.
type logger struct {
	level int
}

func newLogger(level string) *logger {
	switch strings.ToLower(level) {
	case "debug":
		return &logger{level: debugLevel}
	case "info":
		return &logger{level: infoLevel}
	case "warn", "warning":
		return &logger{level: warnLevel}
	default:
		return &logger{level: errorLevel}
	}
}

func (l *logger) log(level int, name string, msg string) {
	if level < l.level {
		return
	}

	_, _ = fmt.Fprintf(os.Stderr, "[souin] %s %s\n", name, msg)
}

func (l *logger) Debug(args ...interface{})  { l.log(debugLevel, "DEBUG", fmt.Sprint(args...)) }
func (l *logger) Info(args ...interface{})   { l.log(infoLevel, "INFO", fmt.Sprint(args...)) }
func (l *logger) Warn(args ...interface{})   { l.log(warnLevel, "WARN", fmt.Sprint(args...)) }
func (l *logger) Error(args ...interface{})  { l.log(errorLevel, "ERROR", fmt.Sprint(args...)) }
func (l *logger) DPanic(args ...interface{}) { l.log(errorLevel, "DPANIC", fmt.Sprint(args...)) }
func (l *logger) Panic(args ...interface{})  { panic(fmt.Sprint(args...)) }
func (l *logger) Fatal(args ...interface{})  { l.log(errorLevel, "FATAL", fmt.Sprint(args...)) }

func (l *logger) Debugf(template string, args ...interface{}) {
	l.log(debugLevel, "DEBUG", fmt.Sprintf(template, args...))
}

func (l *logger) Infof(template string, args ...interface{}) {
	l.log(infoLevel, "INFO", fmt.Sprintf(template, args...))
}

func (l *logger) Warnf(template string, args ...interface{}) {
	l.log(warnLevel, "WARN", fmt.Sprintf(template, args...))
}

func (l *logger) Errorf(template string, args ...interface{}) {
	l.log(errorLevel, "ERROR", fmt.Sprintf(template, args...))
}

func (l *logger) DPanicf(template string, args ...interface{}) {
	l.log(errorLevel, "DPANIC", fmt.Sprintf(template, args...))
}

func (l *logger) Panicf(template string, args ...interface{}) {
	panic(fmt.Sprintf(template, args...))
}

// Fatalf only logs, a plugin must never exit the Traefik process.
func (l *logger) Fatalf(template string, args ...interface{}) {
	l.log(errorLevel, "FATAL", fmt.Sprintf(template, args...))
}

var _ core.Logger = (*logger)(nil)
