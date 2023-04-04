package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/iostrovok/gorouter"
	"github.com/iostrovok/gorouter/cmd/static_video/fileouter"
	"github.com/iostrovok/gorouter/logger/level"
	"github.com/iostrovok/gorouter/static"
)

type TestHandler struct {
	gorouter.RunHandler

	A string
}

func newIH(a string) *TestHandler {
	return &TestHandler{A: a}
}

func (t TestHandler) Run(ctx *gorouter.Context) error {
	u, _ := json.Marshal(ctx.UrlIds())
	_, _ = ctx.WriteString("dddd-" + t.A + "- u: " + string(u) + "//// \n\n")

	ctx.Logger().Add("DDD", "ttt").Error(errors.New("test-error")).Infof("ddd")

	return nil
}

type TestStaticHandler struct {
	gorouter.RunHandler
}

func newStatic() *TestStaticHandler {
	return &TestStaticHandler{}
}

const fileName = "/Users/vladimirlashko/github/videoads/tmp/author/13-20221129103201-585269298002707063"

func (h *TestStaticHandler) Run(ctx *gorouter.Context) error {
	ctx.FastCtx().Response.Header.Set("cache-control", "private, max-age=21299")
	ctx.FastCtx().Response.Header.Set("content-type", "video/mp4")

	ctx.Set("status_code", fasthttp.StatusNotFound)

	content, err := fileouter.New(fileName)
	if err != nil {
		return err
	}

	return static.ServeContent(ctx.FastCtx(), fileName, time.Now(), content)
}

func main() {
	g := gorouter.New().SetServerName("nginx")

	g.SetLogLevel(level.DebugLevel)

	g.Router().Use(gorouter.MethodGetPost, "/", newIH("for MethodGetPost '/'"))
	g.Router().Use(gorouter.MethodGetPost, "/file", newStatic())

	if err := g.StaticFile("data", "/html"); err != nil {
		panic(err)
	}

	fmt.Printf("port 8081\n\n")
	err := g.Run(context.Background(), ":8081")
	fmt.Printf("err: %+v\n\n", err)
}
