package middleware

import (
	"errors"
	"hermyx/pkg/utils/logger"

	"github.com/valyala/fasthttp"
)

type AuthMiddleware struct {
	headerName  string
	expectedVal string
	logger      *logger.Logger
}

func NewAuthMiddleware(logger *logger.Logger, headerName, expectedVal string) *AuthMiddleware {
	return &AuthMiddleware{
		headerName:  headerName,
		expectedVal: expectedVal,
		logger:      logger,
	}
}

func (a *AuthMiddleware) BeforeRequest(ctx *fasthttp.RequestCtx) error {
	if a.expectedVal == "" {
		return nil
	}
	val := string(ctx.Request.Header.Peek(a.headerName))
	if val == "" {
		a.logger.Warn("Auth middleware: missing header " + a.headerName)
		return errors.New("missing auth")
	}
	if val != a.expectedVal {
		a.logger.Warn("Auth middleware: invalid token")
		return errors.New("invalid auth")
	}
	return nil
}

func (a *AuthMiddleware) AfterResponse(ctx *fasthttp.RequestCtx) error {
	// no-op
	return nil
}
