package static

import (
	"io/fs"
	"path"
	"strings"

	"github.com/valyala/fasthttp"
)

// FS converts fSys to a FileSystem implementation,
// for use with FileServer and NewFileTransport.
func FS(fSys fs.FS) FileSystem {
	return ioFS{fSys}
}

// FileServer returns a handler that serves HTTP requests
// with the contents of the file system rooted at root.
//
// As a special case, the returned file server redirects any request
// ending in "/index.html" to the same path, without the final
// "index.html".
//
// To use the operating system's file system implementation,
// use http.Dir:
//
//	http.Handle("/", http.FileServer(http.Dir("/tmp")))
//
// To use an fs.FS implementation, use http.FS to convert it:
//
//	http.Handle("/", http.FileServer(http.FS(fSys)))
func FileServer(root FileSystem) Handler {
	return &fileHandler{root}
}

type Handler interface {
	ServeHTTP(ctx *fasthttp.RequestCtx) error
}

func (f *fileHandler) ServeHTTP(ctx *fasthttp.RequestCtx) error {
	upath := string(ctx.Path())
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		ctx.URI().SetPath(upath)
	}

	// ignore error here
	return serveFile(ctx, f.root, path.Clean(upath), false, false)
}
