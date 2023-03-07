package gorouter

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

type TestHandler struct {
	*Handler

	A string
}

func (t *TestHandler) Init(_ *Context) error {
	return nil
}

func (t *TestHandler) Run(_ *Context) error {
	return nil
}

func newIH(a string) *TestHandler {
	return &TestHandler{A: a}
}

func TestServer_1(t *testing.T) {
	//t.Skip("BBB")

	g := New()

	set := Set("")
	set.Before(newIH("1"), newIH("2"))
	set.After(newIH("9"), newIH("10"))
	g.Add(MethodGetPost, "/private_group", newIH("for MethodGetPost 'private_group'"), set)

	routerA := g.Router()
	routerA.Before(newIH("A1"), newIH("A2"), newIH("A3")) // 3 handlers before
	routerA.Use(MethodPost, "/", newIH("for MethodGetPost '/'"))
	routerA.Use(MethodPost, "/a", newIH("for MethodPost '/a'"))

	routerB := g.Router()
	routerB.Before(newIH("B1"), newIH("B2"), newIH("B3")) // 3 handlers before
	routerB.After(newIH("B9"), newIH("B10"))              // 2 handlers after

	routerB.Use(MethodGet, "/", newIH("for MethodGet '/'"))
	routerB.Use(MethodGet, "/a", newIH("for MethodGet '/a'"))

	ctx, _ := context.WithTimeout(context.Background(), time.Second*1)
	wg, ctxEr := errgroup.WithContext(ctx)
	wg.Go(func() error {
		return g.Run(ctxEr, ":8081")
	})

	res, err := http.Get("http://localhost:8081/")
	fmt.Printf("TreeResult-2: %+v\n\n", res)
	fmt.Printf("err: %+v\n\n", err)
	assert.NotNil(t, res)
	assert.Nil(t, err)

	//cancel()
	assert.Nil(t, wg.Wait())

	assert.Nil(t, g.Server().Shutdown())
}
