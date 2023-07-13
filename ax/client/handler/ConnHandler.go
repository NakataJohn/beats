package handler

import (
	"sync"

	"github.com/go-netty/go-netty"
)

type ConnHandler struct {
	Ctx    netty.HandlerContext
	Gotctx func(netty.HandlerContext)
	_mutex sync.RWMutex
}

func (c ConnHandler) HandleActive(ctx netty.ActiveContext) {
	c._mutex.RLock()
	c.Gotctx(ctx)
	c._mutex.RUnlock()
	ctx.HandleActive()
}
