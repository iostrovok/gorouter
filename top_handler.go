package gorouter

import "sync"

type HandlerSet struct {
	sync.RWMutex

	ID string // for test and debug

	before  []IHandler   // handlers before main handler
	after   []IHandler   // handlers after main handler
	last    ILastHandler // always last handler with error from handlers as parameter
	handler IRunHandler  // main handler
}

func Set(id string) *HandlerSet {
	return &HandlerSet{
		ID:     id,
		before: make([]IHandler, 0),
		after:  make([]IHandler, 0),
	}
}

func (set *HandlerSet) Run(context *Context) error {
	set.RLock()
	defer set.RUnlock()

	// set up init
	context.AddDebugHandleName(set.handler.Name())
	if err := set.handler.Init(context); err != nil {
		return err
	}

	totalBefore := len(set.before)
	for i := 0; i < totalBefore; i++ {
		context.AddDebugHandleName(set.before[i].Name())
		if err := set.before[i].Run(context); err != nil {
			return err
		}

		if context.Stopped() {
			return nil
		}
	}

	if !context.isSkippedMain {
		context.AddDebugHandleName(set.handler.Name())
		if err := set.handler.Run(context); err != nil {
			return err
		}

		if context.Stopped() {
			return nil
		}
	}

	totalAfter := len(set.after)
	for i := 0; i < totalAfter; i++ {
		context.AddDebugHandleName(set.after[i].Name())
		if err := set.after[i].Run(context); err != nil {
			return err
		}

		if context.Stopped() {
			return nil
		}
	}

	return nil
}

func (set *HandlerSet) RunLast(ctx *Context, err error) error {
	set.RLock()
	defer set.RUnlock()

	if set.last == nil {
		return err
	}

	return set.last.Run(ctx, err)
}

func (set *HandlerSet) Last(handler ILastHandler) *HandlerSet {
	set.Lock()
	defer set.Unlock()

	set.last = handler
	return set
}

func (set *HandlerSet) Before(handler ...IHandler) *HandlerSet {
	set.Lock()
	defer set.Unlock()

	set.before = append(set.before, handler...)
	return set
}

func (set *HandlerSet) After(handler ...IHandler) *HandlerSet {
	set.Lock()
	defer set.Unlock()

	set.after = append(set.after, handler...)
	return set
}

func (set *HandlerSet) Use(handler IRunHandler) *HandlerSet {
	set.Lock()
	defer set.Unlock()

	set.handler = handler
	return set
}

func (set *HandlerSet) Clone() *HandlerSet {
	set.Lock()
	defer set.Unlock()

	return &HandlerSet{
		ID:      set.ID,
		before:  set.before[:],
		after:   set.after[:],
		handler: set.handler,
		last:    set.last,
	}
}
