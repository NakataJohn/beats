package handler

import (
	"fmt"
	"sync"

	"github.com/go-netty/go-netty"
)

type ConnHandler struct {
	Ctx    netty.HandlerContext
	Gotctx func(netty.HandlerContext)
	_mutex sync.RWMutex
}

func (c ConnHandler) HandleActive(ctx netty.ActiveContext) {
	fmt.Println("=========>", "服务  连接", ctx.Channel().RemoteAddr())
	c._mutex.RLock()
	// c.Ctx = ctx
	c.Gotctx(ctx)
	c._mutex.RUnlock()
	ctx.HandleActive()
}
