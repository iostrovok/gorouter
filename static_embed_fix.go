package gorouter

import (
	"embed"
	"io/fs"

	"github.com/iostrovok/gorouter/static"
)

type FixStaticHandler struct {
	Handler

	filePath, urlPath string
	emfs              embed.FS
	httpHandler       static.Handler
}

func (h *FixStaticHandler) Init(_ *Context) error {
	return nil
}

func (h *FixStaticHandler) Run(ctx *Context) error {
	old := string(ctx.fastCtx.Path())
	defer func() {
		ctx.fastCtx.URI().SetPath(old)
	}()

	ctx.fastCtx.URI().SetPath(h.filePath)

	ctx.logger.AddDebug("url_path_income", old).
		AddDebug("url_path_internal", h.filePath)

	return h.httpHandler.ServeHTTP(ctx.fastCtx)
}

// FixStaticFileFS works with embed FS for static file
func (server *Server) FixStaticFileFS(filePath, urlPath string, emfs embed.FS, sets ...*HandlerSet) error {
	f, err := fs.Sub(emfs, ".")
	if err != nil {
		return err
	}

	h := &FixStaticHandler{
		urlPath:     urlPath,
		filePath:    filePath,
		emfs:        emfs,
		httpHandler: static.FileServer(static.FS(f)),
	}

	var set *HandlerSet
	if len(sets) > 0 {
		set = sets[0]
	} else {
		set = Set("static for '" + urlPath + "'")
	}

	server.Add(MethodGetPost, urlPath, h, set)

	return nil
}
