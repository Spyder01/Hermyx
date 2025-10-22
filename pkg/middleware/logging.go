package middleware

import (
	"fmt"
	"hermyx/pkg/utils/logger"

	"github.com/valyala/fasthttp"
)

type LoggingMiddleware struct {
	logger *logger.Logger
}

func NewLoggingMiddleware(l *logger.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{logger: l}
}

func (l *LoggingMiddleware) BeforeRequest(ctx *fasthttp.RequestCtx) error {
	l.logger.Info(fmt.Sprintf("Middleware(before) - %s %s", string(ctx.Method()), string(ctx.Path())))
	return nil
}

func (l *LoggingMiddleware) AfterResponse(ctx *fasthttp.RequestCtx) error {
	l.logger.Info(fmt.Sprintf("Middleware(after) - %d %s", ctx.Response.StatusCode(), string(ctx.Path())))
	return nil
}
