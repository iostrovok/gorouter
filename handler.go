package gorouter

type IRunHandler interface {
	Run(ctx *Context) error
	Init(ctx *Context) error
}

type IHandler interface {
	Run(ctx *Context) error
}

type ILastHandler interface {
	Run(ctx *Context, err error) error
}

type Handler struct{}

func (h *Handler) Run(_ *Context, _ error) error { return nil }

type RunHandler struct{}

func (h *RunHandler) Run(_ *Context, _ error) error { return nil }
func (h *RunHandler) Init(_ *Context) error         { return nil }
