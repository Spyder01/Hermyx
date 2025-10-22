package middleware

import "github.com/valyala/fasthttp"

type Middleware interface {
	BeforeRequest(ctx *fasthttp.RequestCtx) error
	AfterResponse(ctx *fasthttp.RequestCtx) error
}

type Chain struct {
	middlewares []Middleware
}

func (c *Chain) Use(middleware *LoggingMiddleware) {
	c.middlewares = append(c.middlewares, middleware)
}

func NewChain(m ...Middleware) *Chain {
	return &Chain{middlewares: m}
}

func (c *Chain) Handle(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {

		for _, mw := range c.middlewares {
			if err := mw.BeforeRequest(ctx); err != nil {
				ctx.Error(err.Error(), fasthttp.StatusForbidden)
				return
			}
		}

		next(ctx)

		for _, mw := range c.middlewares {
			_ = mw.AfterResponse(ctx)
		}
	}
}
