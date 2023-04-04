package text

// TrimString returns s without leading and trailing ASCII space.
func TrimString(s string) string {
	i := 0
	j := len(s)

	for i < j && isASCIISpace(s[i]) {
		i++
	}

	for j > i && isASCIISpace(s[j-1]) {
		j--
	}

	if i > 0 || j != len(s) {
		return s[i:j]
	}

	return s
}

// TrimBytes returns b without leading and trailing ASCII space.
func TrimBytes(b []byte) []byte {
	i := 0
	j := len(b)

	for i < j && isASCIISpace(b[i]) {
		i++
	}

	for j > i && isASCIISpace(b[j-1]) {
		j--
	}

	if i > 0 || j != len(b) {
		return b[i:j]
	}

	return b
}

func isASCIISpace(b byte) bool {
	return b == '\n' || b == '\r' || b == ' ' || b == '\t'
}

func isASCIILetter(b byte) bool {
	b |= 0x20 // make lower case
	return 'a' <= b && b <= 'z'
}
