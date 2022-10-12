package gorouter

import (
	"net/http"

	"github.com/valyala/fasthttp"
)

type Method string

const (
	MethodGet     = fasthttp.MethodGet
	MethodHead    = fasthttp.MethodHead
	MethodPost    = fasthttp.MethodPost
	MethodPut     = fasthttp.MethodPut
	MethodPatch   = fasthttp.MethodPatch
	MethodDelete  = fasthttp.MethodDelete
	MethodConnect = fasthttp.MethodConnect
	MethodOptions = fasthttp.MethodOptions
	MethodTrace   = fasthttp.MethodTrace
	MethodGetPost = fasthttp.MethodGet + http.MethodPost
	MethodGetHead = fasthttp.MethodGet + http.MethodHead
)
