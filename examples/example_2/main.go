package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/iostrovok/gorouter"
	"github.com/iostrovok/gorouter/logger/level"
)

//go:embed data/*
var embedHtml embed.FS

const CookieName = "test_cookie"

type TestHandler struct {
	gorouter.RunHandler

	A string
}

func newIH(a string) *TestHandler {
	return &TestHandler{A: a}
}

func (t TestHandler) Run(ctx *gorouter.Context) error {
	b := ctx.FastCtx().Request.Header.Cookie(CookieName)
	fmt.Printf("cookie-1: %+v\n", string(b))
	cookie, _ := ctx.Cookie(CookieName)
	fmt.Printf("cookie-2: %+v \n", cookie)

	fmt.Printf("cookie: %+v\n", cookie.String())
	fmt.Printf("Name: %+v\n", string(cookie.Key()))
	fmt.Printf("cookie.Value(): '%+v'\n", string(cookie.Value()))
	fmt.Printf("cookie.Expires: %+v\n", cookie.Expire())
	if string(cookie.Value()) == "" {
		ctx.SetCookie("test_cookie", "test", 3600, "", "", false, true)
	}

	u, _ := json.Marshal(ctx.UrlIds())
	_, _ = ctx.WriteString("dddd-" + t.A + "- u: " + string(u) + "//// \n\n")

	ctx.Logger().Add("DDD", "ttt").Error(errors.New("test-error")).Infof("ddd")

	return nil
}

func main() {
	g := gorouter.New().SetServerName("nginx")

	g.SetLogLevel(level.DebugLevel)

	g.Router().
		Use(gorouter.MethodGetPost, "/", newIH("for MethodGetPost '/'")).
		Use(gorouter.MethodGetPost, "/cookie", newIH("for MethodGetPost 'cookie'"))

	if err := g.StaticFileFS("data", "/html", embedHtml); err != nil {
		panic(err)
	}

	if err := g.FixStaticFileFS("/data/favicons/favicon.ico", "/favicon.ico", embedHtml); err != nil {
		panic(err)
	}

	fmt.Printf("port 8081\n\n")
	err := g.Run(context.Background(), ":8081")
	fmt.Printf("err: %+v\n\n", err)
}
