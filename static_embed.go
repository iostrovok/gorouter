package gorouter

import (
	"embed"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/iostrovok/gorouter/static"
)

type StaticHandler struct {
	Handler

	filePath, urlPath string
	emfs              embed.FS
	httpHandler       static.Handler
}

func (h *StaticHandler) Init(_ *Context) error {
	return nil
}

func (h *StaticHandler) Run(ctx *Context) error {
	old := string(ctx.fastCtx.Path())
	defer func() {
		ctx.fastCtx.URI().SetPath(old)
	}()

	u := strings.TrimPrefix(old, "/")
	u = strings.TrimPrefix(u, h.urlPath)
	newU := path.Join(h.filePath, u)
	ctx.fastCtx.URI().SetPath(newU)

	ctx.logger.AddDebug("url_path_income", old).
		AddDebug("url_path_internal", newU)

	return h.httpHandler.ServeHTTP(ctx.fastCtx)
}

// StaticFileFS works with embed FS for static files
func (server *Server) StaticFileFS(filePath, urlPath string, emfs embed.FS, sets ...*HandlerSet) error {
	f, err := fs.Sub(emfs, ".")
	if err != nil {
		return err
	}

	h := &StaticHandler{
		urlPath:     strings.TrimPrefix(urlPath, "/"),
		filePath:    strings.TrimPrefix(filePath, "/"),
		emfs:        emfs,
		httpHandler: static.FileServer(static.FS(f)),
	}

	server.Debugf("static_file sets prefix: from '%s' to '%s'", urlPath, filePath)

	pathReg, err := regexp.Compile("^" + urlPath + "/.*")
	if err != nil {
		return err
	}

	var set *HandlerSet
	if len(sets) > 0 {
		set = sets[0]
	} else {
		set = Set("static for '" + urlPath + "'")
	}
	server.AddReg(MethodGetHead, pathReg, h, set)

	return nil
}
