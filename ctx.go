package gorouter

import (
	"bytes"
	"context"
	"errors"
	"hash/crc64"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"

	"github.com/iostrovok/gorouter/logger"
	"github.com/iostrovok/gorouter/logger/level"
)

var table *crc64.Table

type Cookies []*http.Cookie

func init() {
	table = crc64.MakeTable(crc64.ISO)
}

func (c Cookies) Get(name string) *http.Cookie {
	for i := range c {
		if c[i].Name == name {
			return c[i]
		}
	}

	return &http.Cookie{}
}

type Context struct {
	baseCtx context.Context // global context

	uniqId uint64 // uniq request id, for tests and debug purposes.

	sameSite fasthttp.CookieSameSite
	fastCtx  *fasthttp.RequestCtx

	urlIDs *fasthttp.Args // income  url params

	data map[string]any
	// isStopped doesn't call all next handlers
	isStopped bool
	// isAborted is the same as isStopped but cancel request context too. It's useful for security requests.
	isAborted bool
	// isSkippedMain doesn't call main handler, but calls "before" and "after" handlers
	isSkippedMain bool

	logger *logger.Logger

	eg         *errgroup.Group
	requestCtx context.Context
	cancel     context.CancelFunc

	handleDebugPipeline []string
}

func NewContext(baseCtx context.Context, fastCtx *fasthttp.RequestCtx, args *fasthttp.Args) (*Context, error) {
	cCtx, cancel := context.WithCancel(fastCtx)
	eg, ctx := errgroup.WithContext(cCtx)

	out := &Context{
		fastCtx:             fastCtx,
		requestCtx:          ctx,
		baseCtx:             baseCtx,
		data:                map[string]any{},
		sameSite:            fasthttp.CookieSameSiteDisabled,
		cancel:              cancel,
		urlIDs:              args,
		logger:              logger.New(),
		uniqId:              crc64.Checksum([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)+string(fastCtx.Path())), table),
		eg:                  eg,
		handleDebugPipeline: []string{},
	}

	return out, nil
}

func (ctx *Context) Clone() *Context {
	data := map[string]any{}
	for k := range ctx.data {
		data[k] = ctx.data[k]
	}

	return &Context{
		fastCtx:  ctx.fastCtx,
		baseCtx:  ctx.baseCtx,
		data:     data,
		sameSite: ctx.sameSite,
		urlIDs:   ctx.urlIDs,
	}
}

// Method returns request's method
func (ctx *Context) Method() []byte {
	return ctx.fastCtx.Method()
}

// IsMethod check income method and request's method
func (ctx *Context) IsMethod(method string) bool {
	return method == string(ctx.fastCtx.Method())
}

// Ctx returns context of request
func (ctx *Context) Ctx() context.Context {
	return ctx.requestCtx
}

// Write writes p into response body.
func (ctx *Context) Write(p []byte) (int, error) {
	ctx.fastCtx.Response.AppendBody(p)
	return len(p), nil
}

// WriteString appends s to response body.
func (ctx *Context) WriteString(s string) (int, error) {
	ctx.fastCtx.Response.AppendBodyString(s)
	return len(s), nil
}

func (ctx *Context) SetLogger(lg *logger.Logger) *Context {
	ctx.logger = lg

	return ctx
}

func (ctx *Context) Logger() *logger.Logger {
	return ctx.logger
}

func (ctx *Context) SetLoggerLevel(lvl level.Level) *Context {
	ctx.logger.SetLevel(lvl)

	return ctx
}

func (ctx *Context) LoggerLevel() level.Level {
	return ctx.logger.Level()
}

// RCtx returns "local" context from request.
func (ctx *Context) RCtx() context.Context {
	return ctx.baseCtx
}

func (ctx *Context) FastCtx() *fasthttp.RequestCtx {
	return ctx.fastCtx
}

func (ctx *Context) GlobalContext() context.Context {
	return ctx.baseCtx
}

func (ctx *Context) UrlIds() *fasthttp.Args {
	return ctx.urlIDs
}

var NoCookiesFoundError = errors.New("no cookies found")

func (ctx *Context) Cookie(name string) (*fasthttp.Cookie, error) {
	cookie := &fasthttp.Cookie{}
	b := ctx.fastCtx.Request.Header.Cookie(name)
	if len(b) == 0 {
		return cookie, NoCookiesFoundError
	}
	err := cookie.ParseBytes(b)
	return cookie, err
}

func (ctx *Context) PeekParam(key string) []byte {
	if out := ctx.urlIDs.Peek(key); len(out) > 0 {
		return out
	}

	if out := ctx.fastCtx.QueryArgs().Peek(key); len(out) > 0 {
		return out
	}

	return ctx.fastCtx.PostArgs().Peek(key)
}

func (ctx *Context) PeekStringParam(key string) string {
	return string(ctx.PeekParam(key))
}

func (ctx *Context) PeekBoolParam(key string) bool {
	switch strings.ToLower(ctx.PeekStringParam(key)) {
	case "", "0", "f", "false":
		return false
	}

	return true
}

func (ctx *Context) SqueezeParams() map[string]any {
	out := map[string]any{}

	ctx.fastCtx.PostArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)
		old, find := out[k]
		if !find {
			out[k] = v
			return
		}

		switch t := old.(type) {
		case string:
			out[k] = []string{v, t}
		case []string:
			out[k] = append(t, v)
		}
	})

	return out
}

func (ctx *Context) UniqId() uint64 {
	return ctx.uniqId
}

func (ctx *Context) Set(key string, value any) *Context {
	ctx.data[key] = value

	return ctx
}

func (ctx *Context) Get(key string) any {
	return ctx.data[key]
}

// SetSameSite with cookie
func (ctx *Context) SetSameSite(sameSite fasthttp.CookieSameSite) *Context {
	ctx.sameSite = sameSite

	return ctx
}

// Stop stops calling all next handlers
func (ctx *Context) Stop() *Context {
	ctx.isStopped = true
	ctx.logger.AddDebug("is_stopped", true)

	return ctx
}

// SkipMain skips the calling of main handler, but calls "before" and "after" handlers
func (ctx *Context) SkipMain() *Context {
	ctx.isSkippedMain = true
	ctx.logger.AddDebug("is_skipped_main", true)

	return ctx
}

// Abort is the same as Stop function, but cancel request context too. It's useful for security requests.
func (ctx *Context) Abort() *Context {
	ctx.isAborted = true
	ctx.cancel()

	ctx.logger.AddDebug("is_aborted", true)

	return ctx
}

func (ctx *Context) Aborted() bool {
	return ctx.isAborted
}

func (ctx *Context) Stopped() bool {
	return ctx.isAborted || ctx.isStopped
}

func (ctx *Context) Debugf(format string, data ...any) {
	if ctx.logger.Config().Level() >= level.DebugLevel {
		logger.New().
			SetConfig(ctx.logger.Config()).
			Debugf(format, data...)
	}
}

// splitHostPort separates host and port. If the port is not valid, it returns
// the entire input as host, and it doesn't check the validity of the host.
// Unlike net.SplitHostPort, but per RFC 3986, it requires ports to be numeric.
func splitHost(hostPort []byte) (host []byte) {
	host = hostPort

	if colon := bytes.LastIndexByte(host, ':'); colon != -1 && validOptionalPort(host[colon:]) {
		host = host[:colon]
	}

	if bytes.HasPrefix(host, []byte(`[`)) && bytes.HasSuffix(host, []byte(`]`)) {
		host = host[1 : len(host)-1]
	}

	return
}

// validOptionalPort reports whether port is either an empty string
// or matches /^:\d*$/
func validOptionalPort(port []byte) bool {
	if len(port) == 0 {
		return true
	}

	if port[0] != ':' {
		return false
	}

	for _, b := range port[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}

	return true
}

// SetCookie adds a Set-Cookie header to the ResponseWriter's headers.
func (ctx *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) *Context {
	if path == "" {
		path = "/"
	}

	if domain == "" {
		domain = string(splitHost(ctx.fastCtx.Host()))
	}

	cookie := fasthttp.AcquireCookie()

	cookie.SetDomain(domain)
	cookie.SetPath(path)
	cookie.SetMaxAge(maxAge)
	cookie.SetKey(name)
	cookie.SetKey(name)
	cookie.SetValue(url.QueryEscape(value))
	cookie.SetSecure(secure)
	cookie.SetSameSite(ctx.sameSite)
	cookie.SetHTTPOnly(httpOnly)

	ctx.fastCtx.Response.Header.SetCookie(cookie)

	fasthttp.ReleaseCookie(cookie)

	return ctx
}

func (ctx *Context) EGWait() error {
	return ctx.eg.Wait()
}

func (ctx *Context) EGGo(f func() error) {
	ctx.eg.Go(f)
}

func (ctx *Context) EGTryGo(f func() error) bool {
	return ctx.eg.TryGo(f)
}

func (ctx *Context) EGSetLimit(n int) {
	ctx.eg.SetLimit(n)
}

//// FileFromFS writes the specified file from http.FileSystem into the body stream in an efficient way.
//func (ctx *Context) FileFromFS(filepath string, emfs embed.FS) {
//	r := ctx.request.Clone(ctx.request.Context())
//	r.URL.Path = filepath
//
//	f, err := fs.Sub(emfs, "static")
//	if err != nil {
//		panic(err)
//	}
//
//	http.FileServer(http.FS(f)).ServeHTTP(ctx.writer, r)
//}

// AddDebugHandleName collects the names of the handlers that were called
func (ctx *Context) AddDebugHandleName(name string) *Context {
	if ctx.logger.IsDebug() {
		ctx.handleDebugPipeline = append(ctx.handleDebugPipeline, name)
	}

	return ctx
}

// CalledHandles is a simple getter
func (ctx *Context) CalledHandles() []string {
	if !ctx.logger.IsDebug() {
		return []string{"available in debug mode only"}
	}

	return ctx.handleDebugPipeline
}
