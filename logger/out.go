package logger

import (
	"fmt"
	"os"

	"github.com/iostrovok/gorouter/logger/config"
	"github.com/iostrovok/gorouter/logger/level"
)

func (l *Logger) Log(level level.Level) {
	if level > l.config.Level() {
		return
	}

	logField := l.config.CurrentKey(config.LevelField)
	errorMessageField := l.config.CurrentKey(config.ErrorMessageField)

	l.Lock()
	defer l.Unlock()
	l.Fields[logField] = l.config.Level().String()
	if l.err != nil {
		l.Fields[errorMessageField] = l.err.Error()
	} else {
		l.Fields[errorMessageField] = ""
	}

	_, err := l.config.Writer().Write(l.Fields.Json())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

func (l *Logger) Logf(level level.Level, format string, data ...any) {
	if level > l.config.Level() {
		return
	}

	l.Fields = l.Fields.message(config.MessageField, format, data...)
	l.Log(level)
}

func (l *Logger) Tracef(format string, data ...any) {
	l.Logf(level.TraceLevel, format, data...)
}

func (l *Logger) Printf(format string, data ...any) {
	l.Logf(level.InfoLevel, format, data...)
}

func (l *Logger) Infof(format string, data ...any) {
	l.Logf(level.InfoLevel, format, data...)
}

func (l *Logger) Debugf(format string, data ...any) {
	l.Logf(level.DebugLevel, format, data...)
}

func (l *Logger) Warnf(format string, data ...any) {
	l.Logf(level.WarnLevel, format, data...)
}

func (l *Logger) Panicf(format string, data ...any) {
	l.Logf(level.PanicLevel, format, data...)
}

func (l *Logger) Errorf(format string, data ...any) {
	l.Logf(level.ErrorLevel, format, data...)
}

func (l *Logger) Fatalf(format string, data ...any) {
	l.Logf(level.FatalLevel, format, data...)
}
