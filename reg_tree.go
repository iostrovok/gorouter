package gorouter

import (
	"regexp"

	"github.com/valyala/fasthttp"
)

type RegTree struct {
	Handlers map[Method]map[*regexp.Regexp]*HandlerSet
}

func newRegTree() *RegTree {
	return &RegTree{
		Handlers: map[Method]map[*regexp.Regexp]*HandlerSet{
			MethodGet:     {},
			MethodHead:    {},
			MethodPost:    {},
			MethodPut:     {},
			MethodPatch:   {},
			MethodDelete:  {},
			MethodConnect: {},
			MethodOptions: {},
			MethodTrace:   {},
		},
	}
}

func (rt *RegTree) Add(method Method, path *regexp.Regexp, set *HandlerSet) {
	switch method {
	case MethodGetPost:
		rt.Handlers[MethodGet][path] = set
		rt.Handlers[MethodPost][path] = set
	case MethodGetHead:
		rt.Handlers[MethodGet][path] = set
		rt.Handlers[MethodHead][path] = set
	default:
		rt.Handlers[method][path] = set
	}
}

func (rt *RegTree) Find(method Method, path string, args *fasthttp.Args) TreeResult {
	args.Reset()
	if res, find := rt.Handlers[method]; find {
		for rg, handler := range res {
			if rg.MatchString(path) {
				return TreeResult{
					HandlerSet: handler,
					Find:       true,
				}
			}
		}
	}

	return TreeResult{}
}
