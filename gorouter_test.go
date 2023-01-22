package gorouter

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(ctx context.Context) {
		err := g.Run(ctx, ":8081")
		assert.Nil(t, err)
		wg.Done()
	}(ctx)

	res, err := http.Get("http://localhost:8081/")
	fmt.Printf("TreeResult: %+v\n\n", res)
	fmt.Printf("err: %+v\n\n", err)
	cancel()
	wg.Wait()

	assert.Nil(t, nil)
}
