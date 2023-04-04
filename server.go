package gorouter

import (
	"context"
	"io"
	"os"
	"regexp"

	"github.com/valyala/fasthttp"

	"github.com/iostrovok/gorouter/logger"
	"github.com/iostrovok/gorouter/logger/config"
	"github.com/iostrovok/gorouter/logger/level"
)

type InitCtx func(ctx *Context) error

type Server struct {
	ctx context.Context
	//ln  net.Listener

	srv *fasthttp.Server

	tree    *Tree
	regTree *RegTree

	logConfig *config.Config
	initCtx   InitCtx

	// shutdownTimeOut is max time for shutdown server in millisecond
	shutdownTimeOut int

	serverName string
}

func New() *Server {
	return &Server{
		tree:      newTree(),
		regTree:   newRegTree(),
		logConfig: config.NewConfig(),
	}
}

func (server *Server) Server() *fasthttp.Server {
	return server.srv
}

func (server *Server) SetServer(srv *fasthttp.Server) *Server {
	server.srv = srv
	return server
}

func (server *Server) ServerName() string {
	return server.serverName
}

func (server *Server) SetServerName(name string) *Server {
	server.serverName = name
	return server
}

func (server *Server) ShutdownTimeOut() int {
	return server.shutdownTimeOut
}

func (server *Server) SetShutdownTimeOut(shutdownTimeOut int) *Server {
	server.shutdownTimeOut = shutdownTimeOut
	return server
}

func (server *Server) LoggerWriter() io.Writer {
	return server.logConfig.Writer()
}

func (server *Server) SetLoggerWriter(loggerWriter io.Writer) *Server {
	server.logConfig.SetWriter(loggerWriter)
	return server
}

func (server *Server) SetLogLevel(mode level.Level) *Server {
	server.logConfig.SetLevel(mode)
	return server
}

func (server *Server) LogLevel() level.Level {
	return server.logConfig.Level()
}

func (server *Server) Debugf(format string, data ...any) {
	if server.logConfig.Level() >= level.DebugLevel {
		logger.New().
			SetConfig(server.logConfig).
			Debugf(format, data...)
	}
}

func (server *Server) Router() *Router {
	return newRouter(server)
}

func (server *Server) Tree() *Tree {
	return server.tree
}

func (server *Server) Find(method Method, path string, args *fasthttp.Args) TreeResult {
	if res := server.tree.Find(method, path, args); res.Find {
		return res
	}

	if res := server.regTree.Find(method, path, args); res.Find {
		return res
	}

	return TreeResult{}
}

func resolveAddress(addr []string) string {
	switch len(addr) {
	case 0:
		if port := os.Getenv("PORT"); port != "" {
			return ":" + port
		}
		return ":8080"
	case 1:
		return addr[0]
	default:
		panic("too many parameters")
	}
}

func (server *Server) Add(method Method, route string, handler IRunHandler, set *HandlerSet) *Server {
	server.tree.Add(method, route, set.Clone().Use(handler))

	return server
}

func (server *Server) AddReg(method Method, route *regexp.Regexp, handler IRunHandler, set *HandlerSet) *Server {
	server.regTree.Add(method, route, set.Clone().Use(handler))

	return server
}

func nonEmptySet(set ...*HandlerSet) *HandlerSet {
	if len(set) > 0 {
		return set[0]
	}

	return nil
}

func (server *Server) GetPost(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodGetPost, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Post(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodPost, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Get(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodGet, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Head(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodHead, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Put(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodPut, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Patch(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodPatch, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Delete(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodDelete, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Connect(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodConnect, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Options(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodOptions, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) Trace(route string, handler IRunHandler, set ...*HandlerSet) *Server {
	server.Add(MethodTrace, route, handler, nonEmptySet(set...))

	return server
}

func (server *Server) SesInit(initCtx InitCtx) *Server {
	server.initCtx = initCtx
	return server
}
