package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/iostrovok/gorouter"
)

type TestHandler struct {
	gorouter.RunHandler

	A string
}

func (t *TestHandler) Run(ctx *gorouter.Context) error {
	urlIds, _ := json.Marshal(ctx.UrlIds())
	params, _ := json.Marshal(ctx.FastCtx().QueryArgs())
	c, _ := ctx.Cookie("session")
	cookie, _ := json.Marshal(c)

	_, _ = ctx.Write([]byte("dddd-" + t.A + "- urlIds: " + string(urlIds) +
		"- params: " + string(params) + " - cookie: " + string(cookie) + "//// \n\n"))
	return nil
}

func newIH(a string) *TestHandler {
	return &TestHandler{A: a}
}

func main() {
	g := gorouter.New()

	set := gorouter.Set("")
	set.Before(newIH("1"), newIH("2"))
	set.After(newIH("9"), newIH("10"))
	g.Add(gorouter.MethodGetPost, "/private_group", newIH("for MethodGetPost 'private_group'"), set)

	routerA := g.Router()
	routerA.Before(newIH("A1"), newIH("A2"), newIH("A3")) // 3 handlers before
	routerA.Use(gorouter.MethodPost, "/", newIH("for MethodGetPost '/'"))
	routerA.Use(gorouter.MethodGetPost, "/a", newIH("for MethodPost '/a'"))

	routerB := g.Router()
	routerB.Before(newIH("B1"), newIH("B2"), newIH("B3")) // 3 handlers before
	routerB.After(newIH("B9"), newIH("B10"))              // 2 handlers after

	routerB.Use(gorouter.MethodGet, "/", newIH("for MethodGet '/'"))
	routerB.Use(gorouter.MethodGet, "/b", newIH("for MethodGet '/b'"))

	routerC := g.Router()
	routerC.After(newIH("B9"), newIH("B10")) // 2 handlers after
	routerC.Use(gorouter.MethodGet, "/c", newIH("for MethodGet '/c'"))

	routerP := g.Router()
	routerP.After(newIH("B9"), newIH("B10")) // 2 handlers after
	routerP.Use(gorouter.MethodGet, "/user/:user_id", newIH("for MethodGet '/c'"))
	routerP.Use(gorouter.MethodGet, "/user/:user_id/:action", newIH("for MethodGet '/c'"))

	err := g.Run(context.Background(), ":8081")
	fmt.Printf("err: %+v\n\n", err)
}
