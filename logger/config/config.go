package config

import (
	"io"
	"os"
	"sync"

	"github.com/iostrovok/gorouter/logger/level"
)

// All available fields as constants
const (
	DefaultTimestampFormat = "2006-01-02T15:04:05.999Z"

	MessageField      = "message"       // Error code describing the error. type: keyword
	ErrorMessageField = "error.message" // Error message. type: text
	TimestampField    = "@timestamp"    // Error message. type: text
	LevelField        = "@level"        // level of log. type: text
)

type Config struct {
	sync.RWMutex

	level      level.Level
	writer     io.Writer
	FieldsKeys map[string]string
}

func NewConfig() *Config {
	return &Config{
		// default out
		writer: os.Stdout,

		// default level
		level: level.DebugLevel,

		FieldsKeys: map[string]string{
			MessageField:      MessageField,
			ErrorMessageField: ErrorMessageField,
			TimestampField:    TimestampField,
			LevelField:        LevelField,
		},
	}
}

func (cf *Config) Clone() *Config {
	cf.Lock()
	defer cf.Unlock()

	out := &Config{
		writer:     cf.writer,
		level:      cf.level,
		FieldsKeys: map[string]string{},
	}

	for k := range cf.FieldsKeys {
		out.FieldsKeys[k] = cf.FieldsKeys[k]
	}

	return out
}

func (cf *Config) CurrentKey(key string) string {
	if old, find := cf.FieldsKeys[key]; find {
		return old
	}

	return key
}

func (cf *Config) Level() level.Level {
	return cf.level
}

func (cf *Config) Writer() io.Writer {
	return cf.writer
}

func (cf *Config) SetWriter(writer io.Writer) *Config {
	cf.Lock()
	defer cf.Unlock()

	cf.writer = writer
	return cf
}

func (cf *Config) StdKeys(key, value string) string {
	if key == "" || value == "" {
		return ""
	}

	cf.Lock()
	defer cf.Unlock()

	old, find := cf.FieldsKeys[key]
	if !find {
		return ""
	}

	cf.FieldsKeys[key] = value
	return old
}

func (cf *Config) LevelKey(key string) *Config {
	_ = cf.StdKeys(LevelField, key)
	return cf
}

func (cf *Config) ErrorMessageKey(key string) *Config {
	_ = cf.StdKeys(ErrorMessageField, key)
	return cf
}

func (cf *Config) MessageKey(key string) *Config {
	_ = cf.StdKeys(ErrorMessageField, key)
	return cf
}

func (cf *Config) SetLevel(level level.Level) *Config {
	cf.Lock()
	defer cf.Unlock()

	cf.level = level
	return cf
}
