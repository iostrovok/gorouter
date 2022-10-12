package logger

import (
	"fmt"
	"os"

	json "github.com/json-iterator/go"
)

// Fields type, used to pass to `WithFields`.
type Fields map[string]any

func (f Fields) Json() []byte {
	out, err := json.ConfigCompatibleWithStandardLibrary.Marshal(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return []byte{}
	}

	return append(out, []byte("\n")...)
}

func (f Fields) message(key, format string, data ...any) Fields {
	f[key] = fmt.Sprintf(format, data...)
	return f
}

func (f Fields) Merge(m map[string]any) Fields {
	for key, data := range m {
		f[key] = data
	}

	return f
}
