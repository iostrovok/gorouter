package gorouter

type IRunHandler interface {
	Run(ctx *Context) error
	Init(ctx *Context) error

	// for debug goals only
	Name() string
}

type IHandler interface {
	Run(ctx *Context) error
	Name() string
}

type ILastHandler interface {
	Run(ctx *Context, err error) error
	Name() string
}

type Handler struct{}

func (h *Handler) Run(_ *Context, _ error) error { return nil }
func (h *Handler) Name() string                  { return "handler" }

type RunHandler struct{}

func (h *RunHandler) Run(_ *Context, _ error) error { return nil }
func (h *RunHandler) Init(_ *Context) error         { return nil }
func (h *RunHandler) Name() string                  { return "run-handler" }
