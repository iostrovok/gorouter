package main

import (
	"context"
	"fmt"
	"time"

	"github.com/iostrovok/gorouter"
)

type TestHandler struct {
	gorouter.RunHandler

	A string
}

func (t *TestHandler) Run(ctx *gorouter.Context) error {
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

	// 3 handlers before
	routerA := g.Router().Before(newIH("A1"), newIH("A2"), newIH("A3"))
	routerA.Use(gorouter.MethodPost, "/", newIH("for MethodGetPost '/'")).
		Use(gorouter.MethodPost, "/a", newIH("for MethodPost '/a'"))

	// 3 handlers before, 2 handlers after
	g.Router().
		Before(newIH("B1"), newIH("B2"), newIH("B3")).
		After(newIH("B9"), newIH("B10")).
		Use(gorouter.MethodGet, "/", newIH("for MethodGet '/'")).
		Use(gorouter.MethodGet, "/a", newIH("for MethodGet '/a'"))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()

	err := g.Run(ctx, ":8081")
	fmt.Printf("TreeResult: %+v\n\n", err)
}
