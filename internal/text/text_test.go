package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var spaceData = map[string]string{
	"asd":                     "asd",
	"   ":                     "",
	"\n":                      "",
	"   q\n\r":                "q",
	"  ":                      "",
	"  Super header \n ":      "Super header",
	"\r\t  Super\nheader \n ": "Super\nheader",
	"\r \t   Super\nheader":   "Super\nheader",
}

func TestTrimString(t *testing.T) {
	for in, out := range spaceData {
		assert.Equal(t, out, TrimString(in), in+" => "+out)
	}
}

func TestTrimBytes(t *testing.T) {
	for in, out := range spaceData {
		assert.Equal(t, out, string(TrimBytes([]byte(in))), in+" => "+out)
	}
}
