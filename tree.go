package gorouter

import (
	"strings"
	"sync"

	"github.com/valyala/fasthttp"
)

type Tree struct {
	Top *Node
}

type Node struct {
	sync.RWMutex

	Children []*Node

	Path     string
	UrlId    string
	Handlers map[Method]*HandlerSet
}

func newTree() *Tree {
	return &Tree{
		Top: &Node{
			Children: []*Node{},
			Path:     "/",
			Handlers: map[Method]*HandlerSet{},
		},
	}
}

func nextNodes(path, urlId string, children []*Node) []*Node {
	if urlId != "" {
		return children[:]
	}

	for j := range children {
		if children[j].Path == path {
			return []*Node{children[j]}
		}
	}

	return []*Node{}
}

type TreeResult struct {
	HandlerSet *HandlerSet
	Find       bool
}

func splitPath(path string) []string {
	path = strings.TrimSpace("/" + strings.Trim(strings.TrimSpace(path), "/"))
	if path == "/" || path == "" {
		return []string{"/"}
	}

	out := strings.Split(path, "/")
	out[0] = "/"

	return out
}

func addMethod(method Method, handlers map[Method]*HandlerSet, set *HandlerSet) map[Method]*HandlerSet {
	switch method {
	case MethodGetPost:
		handlers[MethodPost] = set
		handlers[MethodGet] = set
	case MethodGetHead:
		handlers[MethodGet] = set
		handlers[MethodHead] = set
	default:
		handlers[method] = set
	}

	return handlers
}

type nextNode struct {
	node *Node
	path []string
	args *fasthttp.Args
}

func (t *Tree) Find(method Method, path string, inOutArgs *fasthttp.Args) TreeResult {
	inOutArgs.Reset()
	pathsIn := splitPath(path)

	if len(pathsIn) == 1 {
		t.Top.RLock()
		defer t.Top.RUnlock()

		// "/" case
		if handlers, find := t.Top.Handlers[method]; find && handlers != nil {
			return TreeResult{
				HandlerSet: handlers,
				Find:       find,
			}
		}

		return TreeResult{}
	}

	nodes := []*nextNode{{
		node: t.Top,
		path: pathsIn,
		args: &fasthttp.Args{},
	}}

	allArgs := []*fasthttp.Args{nodes[0].args}
	defer func() {
		for i := range allArgs {
			fasthttp.ReleaseArgs(allArgs[i])
		}
	}()

	nodeId := 0
	for {
		if nodeId >= len(nodes) {
			return TreeResult{}
		}

		node := nodes[nodeId].node
		paths := nodes[nodeId].path
		urlArgs := nodes[nodeId].args
		nodeId++

		if node.UrlId != "" {
			urlArgs.Add(node.UrlId, paths[0])
		} else if node.Path != paths[0] {
			return TreeResult{}
		}

		if len(paths) == 1 {
			node.RLock()
			handlers, find := node.Handlers[method]
			node.RUnlock()

			if find && handlers != nil {
				// copy inOutArgs
				urlArgs.VisitAll(func(key, value []byte) {
					inOutArgs.AddBytesKV(key, value)
				})

				return TreeResult{
					HandlerSet: handlers,
					Find:       true,
				}
			}

			return TreeResult{}
		}

		for i := range node.Children {
			if node.Children[i].Path == paths[1] || node.Children[i].UrlId != "" {
				allArgs = cloneArgs(urlArgs, allArgs)
				nodes = append(nodes, &nextNode{
					node: node.Children[i],
					path: paths[1:],
					args: allArgs[len(allArgs)-1],
				})
			}
		}
	}
}

func cloneArgs(urlArgs *fasthttp.Args, allArgs []*fasthttp.Args) []*fasthttp.Args {
	newArgs := fasthttp.AcquireArgs()
	urlArgs.VisitAll(func(key, value []byte) {
		newArgs.AddBytesKV(key, value)
	})

	allArgs = append(allArgs, newArgs)
	return allArgs
}

func (t *Tree) Add(method Method, path string, set *HandlerSet) {
	t.Top.Lock()
	defer t.Top.Unlock()

	paths := splitPath(path)
	last := len(paths) - 1
	if last == 0 {
		t.Top.Handlers = addMethod(method, t.Top.Handlers, set)
		return
	}

	t._add(t.Top, method, paths[1:], set)
}

func getUrlID(p string) (string, string) {
	if strings.HasPrefix(p, ":") {
		return "", strings.TrimLeft(p, ":")
	}
	return p, ""
}

func (t *Tree) _add(node *Node, method Method, paths []string, set *HandlerSet) {
	next := nextNodes(paths[0], node.UrlId, node.Children)

	// final point, need to set up HandlerSet
	if len(paths) == 1 {
		if len(next) == 0 {
			path, urlId := getUrlID(paths[0])
			node.Children = append(node.Children, &Node{
				UrlId:    urlId,
				Path:     path,
				Children: []*Node{},
				Handlers: addMethod(method, map[Method]*HandlerSet{}, set),
			})
		} else {
			for i := range next {
				next[i].Handlers = addMethod(method, next[i].Handlers, set)
			}
		}

		return
	}

	if len(next) == 0 {
		for _, p := range paths {
			path, urlId := getUrlID(p)
			n := &Node{
				UrlId:    urlId,
				Path:     path,
				Children: []*Node{},
				Handlers: map[Method]*HandlerSet{},
			}
			node.Children = append(node.Children, n)
			node = n
		}

		node.Handlers = addMethod(method, node.Handlers, set)
		return
	}

	for i := range next {
		t._add(next[i], method, paths[1:], set)
	}
}

func (t *Tree) String() string {
	return t.Top.print("")
}

func (node *Node) print(tab string) string {
	out := tab + "Path: " + node.Path + ", UrlId: " + node.UrlId + " => "
	for m, h := range node.Handlers {
		out += "/" + string(m) + "[" + h.ID + "]"
	}
	out += "\n"

	tab += "\t"
	for i := range node.Children {
		out += node.Children[i].print(tab)
	}

	return out
}
