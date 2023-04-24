package logger

import (
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/iostrovok/gorouter/logger/config"
	"github.com/iostrovok/gorouter/logger/level"
)

type Logger struct {
	sync.RWMutex
	config *config.Config

	err    error
	Fields Fields
}

func New() *Logger {
	return &Logger{
		Fields: map[string]any{
			config.TimestampField: time.Now().Format(config.DefaultTimestampFormat),
		},

		// default config
		config: config.NewConfig(),
	}
}

func (l *Logger) SetConfig(cf *config.Config) *Logger {
	n := cf.Clone()

	l.Lock()
	defer l.Unlock()

	l.config = n.Clone()
	return l
}

func (l *Logger) Config() *config.Config {
	l.Lock()
	defer l.Unlock()

	return l.config.Clone()
}

func (l *Logger) Clone() *Logger {
	out := &Logger{
		Fields: map[string]any{},
		config: l.config.Clone(),
	}

	l.Lock()
	defer l.Unlock()

	for k := range l.Fields {
		out.Fields[k] = l.Fields[k]
	}

	return out
}

func (l *Logger) Add(key string, value any) *Logger {
	l.Lock()
	defer l.Unlock()

	l.Fields[key] = value
	return l
}

func (l *Logger) AddDebug(key string, value any) *Logger {
	l.Lock()
	defer l.Unlock()

	if l.config.Level() >= level.DebugLevel {
		l.Fields[key] = value
	}

	return l
}

func (l *Logger) Merge(m map[string]any) *Logger {
	l.Lock()
	defer l.Unlock()

	l.Fields = l.Fields.Merge(m)
	return l
}

func (l *Logger) Writer(writer io.Writer) *Logger {
	l.config.SetWriter(writer)
	return l
}

func (l *Logger) IsDebug() bool {
	return l.config.Level() == level.DebugLevel
}

func (l *Logger) Level(lvl level.Level) *Logger {
	l.config.SetLevel(lvl)
	return l
}

func (l *Logger) StdKeys(key, value string) *Logger {
	oldField := l.config.StdKeys(key, value)
	if oldField == "" {
		return l
	}

	l.Lock()
	defer l.Unlock()

	if _, find := l.Fields[oldField]; find && oldField != key {
		l.Fields[value] = l.Fields[oldField]
		delete(l.Fields, oldField)
	}

	return l
}

func (l *Logger) LevelKey(key string) *Logger {
	return l.StdKeys(config.LevelField, key)
}

func (l *Logger) ErrorMessageKey(key string) *Logger {
	return l.StdKeys(config.ErrorMessageField, key)
}

func (l *Logger) MessageKey(key string) *Logger {
	return l.StdKeys(config.ErrorMessageField, key)
}

func (l *Logger) Error(err error) *Logger {
	if err == nil {
		return l
	}

	l.Lock()
	defer l.Unlock()

	if l.err == nil {
		l.err = err
	} else {
		l.err = errors.Wrap(l.err, err.Error())
	}

	return l
}
