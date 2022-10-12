package gorouter

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/iostrovok/gorouter/static"
)

type StaticFileHandler struct {
	Handler

	filePath, urlPath string
	httpHandler       static.Handler
}

func (h *StaticFileHandler) Init(_ *Context) error {
	return nil
}

func (h *StaticFileHandler) Run(ctx *Context) error {
	old := string(ctx.fastCtx.Path())
	defer func() {
		ctx.fastCtx.URI().SetPath(old)
	}()

	newU := strings.TrimPrefix(old, "/")
	newU = strings.TrimPrefix(newU, h.urlPath)
	ctx.fastCtx.URI().SetPath(newU)

	ctx.logger.AddDebug("url_path_income", old).AddDebug("url_path_internal", newU)

	return h.httpHandler.ServeHTTP(ctx.fastCtx)
}

// StaticFile works with simple static files
func (server *Server) StaticFile(filePath, urlPath string, sets ...*HandlerSet) error {
	h := &StaticFileHandler{
		urlPath:     strings.TrimPrefix(urlPath, "/"),
		filePath:    strings.TrimPrefix(filePath, "/"),
		httpHandler: static.FileServer(static.FS(os.DirFS(filePath))),
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

	fmt.Printf("MethodGetHead: %+v\n\n", MethodGetHead)
	fmt.Printf("pathReg: %+v\n\n", pathReg)
	fmt.Printf("h: %+v\n\n", h)
	fmt.Printf("set: %+v\n\n", set)
	server.AddReg(MethodGetHead, pathReg, h, set)

	return nil
}
