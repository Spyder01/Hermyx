package middleware

import (
	"hermyx/pkg/utils/logger"

	"github.com/valyala/fasthttp"
)

type TransformMiddleware struct {
	logger *logger.Logger
}

func NewTransformMiddleware(logger *logger.Logger) *TransformMiddleware {
	return &TransformMiddleware{logger: logger}
}

func (t *TransformMiddleware) BeforeRequest(ctx *fasthttp.RequestCtx) error {
	if len(ctx.Request.Header.Peek("X-Forwarded-For")) == 0 {
		ctx.Request.Header.Set("X-Forwarded-For", ctx.RemoteAddr().String())
		t.logger.Debug("Transform middleware: set X-Forwarded-For")
	}
	return nil
}

func (t *TransformMiddleware) AfterResponse(ctx *fasthttp.RequestCtx) error {
	ctx.Response.Header.Set("X-Hermyx-Middleware", "transformed")
	return nil
}
