package gorouter

import (
	"regexp"
	"sync"
)

type Router struct {
	sync.RWMutex

	server *Server
	before []IHandler
	after  []IHandler
	last   ILastHandler
}

func newRouter(server *Server) *Router {
	out := &Router{
		server: server,
		before: make([]IHandler, 0),
		after:  make([]IHandler, 0),
	}

	return out
}

func (router *Router) Last(handler ILastHandler) *Router {
	router.Lock()
	defer router.Unlock()

	router.last = handler
	return router
}

func (router *Router) GetBefore() []IHandler {
	return router.before
}

func (router *Router) GetAfter() []IHandler {
	return router.after
}

func (router *Router) First(h IHandler) *Router {
	router.Lock()
	defer router.Unlock()

	router.before = append([]IHandler{h}, router.before...)

	return router
}

func (router *Router) Before(h ...IHandler) *Router {
	router.Lock()
	defer router.Unlock()

	router.before = h

	return router
}

func (router *Router) After(h ...IHandler) *Router {
	router.Lock()
	defer router.Unlock()

	router.after = h

	return router
}

func (router *Router) LastAfter(h IHandler) *Router {
	router.Lock()
	defer router.Unlock()

	router.after = append(router.after, h)

	return router
}

func (router *Router) Use(method Method, route string, handler IRunHandler) *Router {
	router.Lock()
	defer router.Unlock()

	set := Set("").After(router.after...).Before(router.before...).Last(router.last).Use(handler)
	router.server.tree.Add(method, route, set)

	return router
}

func (router *Router) UseReg(method Method, route *regexp.Regexp, handler IRunHandler) *Router {
	router.Lock()
	defer router.Unlock()

	set := Set("").After(router.after...).Before(router.before...).Last(router.last).Use(handler)
	router.server.regTree.Add(method, route, set)

	return router
}

func (router *Router) GetPost(route string, handler IRunHandler) *Router {
	router.Use(MethodGetPost, route, handler)

	return router
}

func (router *Router) Post(route string, handler IRunHandler) *Router {
	router.Use(MethodPost, route, handler)

	return router
}

func (router *Router) Get(route string, handler IRunHandler) *Router {
	router.Use(MethodGet, route, handler)

	return router
}

func (router *Router) Head(route string, handler IRunHandler) *Router {
	router.Use(MethodHead, route, handler)

	return router
}

func (router *Router) Put(route string, handler IRunHandler) *Router {
	router.Use(MethodPut, route, handler)

	return router
}

func (router *Router) Patch(route string, handler IRunHandler) *Router {
	router.Use(MethodPatch, route, handler)

	return router
}

func (router *Router) Delete(route string, handler IRunHandler) *Router {
	router.Use(MethodDelete, route, handler)

	return router
}

func (router *Router) Connect(route string, handler IRunHandler) *Router {
	router.Use(MethodConnect, route, handler)

	return router
}

func (router *Router) Options(route string, handler IRunHandler) *Router {
	router.Use(MethodOptions, route, handler)

	return router
}

func (router *Router) Trace(route string, handler IRunHandler) *Router {
	router.Use(MethodTrace, route, handler)

	return router
}

func (router *Router) Clone() *Router {
	out := &Router{
		server: router.server,
		last:   router.last,
		before: router.before[:],
		after:  router.after[:],
	}

	return out
}
