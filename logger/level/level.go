package level

import (
	"fmt"
	"strings"
)

// Level type
type Level int

// These are the different logging levels. You can set the logging level to log
// on your instance of logger, obtained with `logrus.New()`.
const (
	MaxLevelNumber = 6

	// PanicLevel level, the highest level of severity. Logs and then calls panic with the message passed to Debug, Info, ...
	PanicLevel Level = 0
	// FatalLevel level. Logs and then calls `logger.Exit(1)`. It will exit even if the logging level is set to Panic.
	FatalLevel Level = 1
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel Level = 2
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel Level = 3
	// InfoLevel level. General operational entries about what's going on inside the application.
	InfoLevel Level = 4
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel Level = 5
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel Level = 6
)

// A constant provides all logging levels
var AllLevels = []Level{
	PanicLevel,
	FatalLevel,
	ErrorLevel,
	WarnLevel,
	InfoLevel,
	DebugLevel,
	TraceLevel,
}

var AllLevelsString = []string{
	"panic",
	"fatal",
	"error",
	"warning",
	"info",
	"debug",
	"trace",
}

// String converts the Level to a string. E.g. PanicLevel becomes "panic".
func (level Level) String() string {
	if b, err := level.Byte(); err == nil {
		return string(b)
	}

	return "unknown"
}

// Parse takes a string level and returns the Logrus log level constant.
func Parse(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	}

	return -1, fmt.Errorf("not a valid logrus SetLevel: %q", lvl)
}

func (level Level) Byte() ([]byte, error) {
	if MaxLevelNumber >= level && level > -1 {
		return []byte(AllLevelsString[level]), nil
	}

	return nil, fmt.Errorf("not a valid logrus level %d", level)
}
