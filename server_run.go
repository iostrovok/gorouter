package gorouter

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"

	"github.com/iostrovok/gorouter/logger"
)

func (server *Server) Run(ctx context.Context, add string, addr ...string) error {
	address := resolveAddress(append([]string{add}, addr...))
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	errGroup, errCtx := errgroup.WithContext(ctx)
	server.ctx = errCtx

	if server.srv == nil {
		server.srv = &fasthttp.Server{}
	}

	server.srv.Handler = func(requestCtx *fasthttp.RequestCtx) {
		server.ServeHTTP(requestCtx)
	}

	if server.serverName != "" {
		server.srv.Name = server.serverName
	}

	errGroup.Go(func() error {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
		select {
		case <-ctx.Done():
			return errors.Wrap(server.srv.Shutdown(), context.Canceled.Error())
		case sig := <-ch:
			return errors.Wrap(server.srv.Shutdown(), "server shutdown: "+sig.String())
		}
	})

	errGroup.Go(func() error {
		// We received an interrupt signal, shut down.
		return server.srv.Serve(ln)
	})

	return errGroup.Wait()
}

func (server *Server) ServeHTTP(fastCtx *fasthttp.RequestCtx) {
	if _, err := server.runHTTP(fastCtx); err != nil {
		log.Println(err.Error())
	}
}

func (server *Server) runHTTP(fastCtx *fasthttp.RequestCtx) (*Context, error) {
	urlIDsArgs := fasthttp.AcquireArgs()
	defer fasthttp.ReleaseArgs(urlIDsArgs)

	path := string(fastCtx.Path())
	set := server.Find(Method(fastCtx.Method()), path, urlIDsArgs)
	if !set.Find {
		// TODO not found handler
		fastCtx.Error("not found", fasthttp.StatusNotFound)
		return nil, errors.New("path '" + path + "'+not found")
	}

	ctx, err := NewContext(server.ctx, fastCtx, urlIDsArgs)
	if err != nil {
		fastCtx.Error("not found", fasthttp.StatusBadRequest)
		return nil, errors.New("path '" + path + "': " + err.Error())
	}

	// add to context additional data
	if server.initCtx != nil {
		if err := server.initCtx(ctx); err != nil {
			return nil, errors.New("path '" + path + "': " + err.Error())
		}
	}

	lg := logger.New().SetConfig(server.logConfig)
	ctx.SetLogger(lg).SetLoggerLevel(server.logConfig.Level())

	err = set.HandlerSet.Run(ctx)
	if ctx.IsAborted {
		return ctx, err
	}

	if egErr := ctx.EGWait(); err == nil {
		err = egErr
	} else if err != nil && egErr != nil {
		err = errors.Wrap(err, egErr.Error())
	}

	err = set.HandlerSet.RunLast(ctx, err)

	return ctx, err
}
