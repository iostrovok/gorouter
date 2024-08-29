package gorouter

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"

	"github.com/iostrovok/gorouter/logger"
)

// Run starts the server. it the main function of the server.
func (server *Server) Run(ctx context.Context, add string, addr ...string) error {
	if server.listener == nil {
		address := resolveAddress(append([]string{add}, addr...))
		ln, err := net.Listen("tcp", address)
		if err != nil {
			return err
		}
		server.listener = ln
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
		case <-errCtx.Done():
			// global (main) context is done. Stop.
			var err error
			if server.shutdownTimeOut > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), //nolint:govet
					time.Duration(server.shutdownTimeOut)*time.Millisecond)
				err = server.srv.ShutdownWithContext(ctx)
				cancel()
			} else {
				err = server.srv.Shutdown()
			}
			return errors.Wrap(err, context.Canceled.Error())
		case sig := <-ch:
			// We received an interrupt signal, shut down.
			return errors.Wrap(server.srv.ShutdownWithContext(ctx), "server shutdown: "+sig.String())
		}
	})

	errGroup.Go(func() error {
		if server.certFile != "" && server.keyFile != "" {
			cert, err := tls.LoadX509KeyPair(server.certFile, server.keyFile)
			if err != nil {
				return errors.Errorf("cannot load TLS certificate from certFile=%q, keyFile=%q: %v", server.certFile, server.keyFile, err)
			}

			serverTLSConfig := &tls.Config{
				Certificates:             []tls.Certificate{cert},
				PreferServerCipherSuites: true,
				CurvePreferences:         []tls.CurveID{},
				//ServerName:
			}

			server.listener = tls.NewListener(server.listener, serverTLSConfig)
		}

		return server.srv.Serve(server.listener)
	})

	return errGroup.Wait()
}

func printError(fastCtx *fasthttp.RequestCtx, err error) {
	if err != nil {
		log.Printf("[%s] %s\n", fastCtx.RemoteAddr().String(), err.Error())
	}
}

func (server *Server) ServeHTTP(fastCtx *fasthttp.RequestCtx) {
	if server.baseAuth.Use() && server.baseAuth.checkAccess(fastCtx) {
		success, err := server.baseAuth.Check(fastCtx)
		printError(fastCtx, err)

		if !success {
			return
		}
	}

	_, err := server.runHTTP(fastCtx)
	printError(fastCtx, err)
}

func (server *Server) runHTTP(fastCtx *fasthttp.RequestCtx) (*Context, error) {
	urlIDsArgs := fasthttp.AcquireArgs()
	defer fasthttp.ReleaseArgs(urlIDsArgs)

	path := string(fastCtx.Path())
	set := server.Find(Method(fastCtx.Method()), path, urlIDsArgs)
	if !set.Find {
		// TODO not found handler
		fastCtx.Error("not found", fasthttp.StatusNotFound)
		return nil, errors.New("path '" + path + "' not found")
	}

	ctx, err := NewContext(server.ctx, fastCtx, urlIDsArgs)
	defer ctx.cancel()
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
	if ctx.isAborted {
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
