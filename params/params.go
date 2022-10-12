package params

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value []string
}

type Params []Param

func (p Params) Get(name string) ([]string, bool) {
	for _, entry := range p {
		if entry.Key == name {
			return entry.Value, true
		}
	}
	return []string{}, false
}

func (p Params) ByName(name string) string {
	va, find := p.Get(name)
	if find {
		return va[0]
	}
	return ""
}
