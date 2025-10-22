package engine

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"hermyx/pkg/utils/fs"
	"hermyx/pkg/utils/regex"

	"github.com/valyala/fasthttp"
)

func (engine *HermyxEngine) compileRoutes() {
	engine.compiledRoutes = []compiledRoute{}
	for i := range engine.config.Routes {
		route := &engine.config.Routes[i]

		cr := compiledRoute{
			Route:       route,
			PathPattern: regexp.MustCompile(route.Path),
		}

		if len(route.Include) > 0 {
			cr.IncludeRegex = regex.CombinePattenrs(route.Include)
		}
		if len(route.Exclude) > 0 {
			cr.ExcludeRegex = regex.CombinePattenrs(route.Exclude)
		}

		route.Cache = engine.cacheManager.Resolve(engine.config.Cache, route.Cache)
		engine.compiledRoutes = append(engine.compiledRoutes, cr)
	}
}

func (engine *HermyxEngine) Run() {
	addr := fmt.Sprintf(":%d", engine.config.Server.Port)
	engine.logger.Info(fmt.Sprintf("Hermyx engine starting on %s...", addr))

	server := &fasthttp.Server{
		Handler:          engine.MiddlewareChain.Handle(engine.handleRequest),
		MaxConnsPerIP:    0,
		DisableKeepalive: false,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(addr); err != nil {
			engine.logger.Error(fmt.Sprintf("Fatal server error: %v", err))
			os.Exit(1)
		}
	}()

	err := engine.storePid()
	if err != nil {
		engine.logger.Error(fmt.Sprintf("Unable to store program information due to %v", err))
	}

	<-stop

	engine.logger.Info("Shutdown signal received. Cleaning up...")

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(); err != nil {
		engine.logger.Error("Error during shutdown: " + err.Error())
	}

	err = engine.cleanup()
	if err != nil {
		engine.logger.Error(fmt.Sprintf("Unable to remove program information due to %v", err))
	}

	engine.logger.Info("Hermyx shut down gracefully.")
	engine.logger.Close()
}

func (engine *HermyxEngine) handleRequest(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	method := strings.ToLower(string(ctx.Method()))
	engine.logger.Info(fmt.Sprintf("Incoming request - Method: %s, Path: %s", method, path))

	cr, matched := engine.matchRoute(path, method)
	if !matched {
		engine.logger.Info(fmt.Sprintf("No route matched for %s %s; proxying raw", method, path))
		if err := engine.fallbackProxy(ctx); err != nil {
			engine.logger.Error("Fallback proxy error: " + err.Error())
			ctx.Error("Fallback proxy error: "+err.Error(), fasthttp.StatusBadGateway)
		}
		return
	}

	if cr.Route.Cache.KeyConfig == nil {
		engine.logger.Error("Cache KeyConfig is nil for route " + cr.Route.Path)
		ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
		return
	}

	key := engine.cacheManager.GetKey(cr.Route.Cache.KeyConfig, ctx)
	engine.logger.Debug(fmt.Sprintf("Cache key generated: %s", key))

	if cr.Route.Cache.Enabled {
		if hit := engine.handleCache(ctx, cr, key); hit {
			return
		}
	}

	if err := engine.proxyRequest(ctx, cr); err != nil {
		engine.logger.Error(fmt.Sprintf("Proxy error for %s %s: %v", method, path, err))
		ctx.Error("Proxy error: "+err.Error(), fasthttp.StatusBadGateway)
		return
	}

	engine.cacheResponse(cr, key, ctx)
}

func (engine *HermyxEngine) matchRoute(path, method string) (*compiledRoute, bool) {
	for i := range engine.compiledRoutes {
		cr := &engine.compiledRoutes[i]

		if !cr.PathPattern.MatchString(path) {
			engine.logger.Debug(fmt.Sprintf("Route %q skipped: path pattern mismatch for %s", cr.Route.Path, path))
			continue
		}

		if cr.IncludeRegex != nil && !cr.IncludeRegex.MatchString(path) {
			engine.logger.Debug(fmt.Sprintf("Route %q skipped: include regex does not match path %s", cr.Route.Path, path))
			continue
		}

		if cr.ExcludeRegex != nil && cr.ExcludeRegex.MatchString(path) {
			engine.logger.Debug(fmt.Sprintf("Route %q skipped: exclude regex matches path %s", cr.Route.Path, path))
			continue
		}

		// Check excluded methods
		if cr.Route.Cache.KeyConfig != nil {
			for _, excludedMethod := range cr.Route.Cache.KeyConfig.ExcludeMethods {
				if strings.ToLower(excludedMethod) == method {
					engine.logger.Info(fmt.Sprintf("Request method %s excluded for route %s", method, cr.Route.Path))
					return nil, false
				}
			}
		}

		return cr, true
	}
	return nil, false
}

func (engine *HermyxEngine) handleCache(ctx *fasthttp.RequestCtx, cr *compiledRoute, key string) bool {
	res, exists, err := engine.cacheManager.Get(key)
	if err != nil {
		engine.logger.Error(fmt.Sprintf("Error while accessing the cache: %s", err.Error()))
		return false
	}

	if exists {
		engine.logger.Info(fmt.Sprintf("Cache HIT for key %s (path %s)", key, string(ctx.Path())))
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Response.Header.Set("X-Hermyx-Cache", "HIT")
		ctx.SetBody(res)
		return true
	}
	engine.logger.Info(fmt.Sprintf("Cache MISS for key %s (path %s)", key, string(ctx.Path())))
	return false
}

func (engine *HermyxEngine) proxyRequest(ctx *fasthttp.RequestCtx, cr *compiledRoute) error {
	target := cr.Route.Target
	client := engine.getClientForTarget(target)

	engine.logger.Info(fmt.Sprintf("Proxying request %s %s to backend %s", string(ctx.Method()), string(ctx.Path()), target))
	return client.Do(&ctx.Request, &ctx.Response)
}

func (engine *HermyxEngine) cacheResponse(cr *compiledRoute, key string, ctx *fasthttp.RequestCtx) {
	status := ctx.Response.StatusCode()
	if status >= 200 && status < 300 {
		body := ctx.Response.Body()
		if uint64(len(body)) <= cr.Route.Cache.MaxContentSize {
			cacheTtl := cr.Route.Cache.Ttl
			engine.cacheManager.Set(key, append([]byte(nil), body...), cacheTtl)
			engine.logger.Info(fmt.Sprintf("Cached response for key %s with TTL %s", key, cacheTtl.String()))
		} else {
			engine.logger.Info(fmt.Sprintf("Response size %d exceeds max cache size %d; skipping cache for key %s", len(body), cr.Route.Cache.MaxContentSize, key))
		}
	} else {
		engine.logger.Debug(fmt.Sprintf("Not caching response for key %s due to status %d", key, status))
	}
}

func (engine *HermyxEngine) fallbackProxy(ctx *fasthttp.RequestCtx) error {
	host := string(ctx.Host())
	if host == "" {
		engine.logger.Warn("No Host header present on unmatched request; returning 404")
		ctx.Error("No route matched and no Host header present", fasthttp.StatusNotFound)
		return nil
	}

	client := engine.getClientForTarget(host)
	engine.logger.Info(fmt.Sprintf("Fallback proxying to %s", host))
	return client.Do(&ctx.Request, &ctx.Response)
}

func (engine *HermyxEngine) storePid() error {
	engine.logger.Info("Storing program id information...")

	storageDir := engine.config.Storage.Path

	path := filepath.Join(storageDir, "hermyx.pid")

	err := fs.EnsureDir(storageDir)
	if err != nil {
		engine.logger.Error(fmt.Sprintf("Unable to create program storage path due to %v", err))
		return err
	}

	err = os.WriteFile(path, []byte(fmt.Sprintf("%d", engine.pid)), 0o644)
	if err != nil {
		engine.logger.Error(fmt.Sprintf("Unable to store program id due to %v", err))
		return err
	}

	engine.logger.Info(fmt.Sprintf("Stored program id information at %s", path))
	return nil
}

func (engine *HermyxEngine) cleanup() error {
	var err error = nil

	err = engine.cacheManager.Close()
	if err != nil {
		engine.logger.Error(fmt.Sprintf("Failed to close the cache due to: %w", err))
	}
	engine.logger.Info("Cache closed")

	pidFile := filepath.Join(engine.config.Storage.Path, "hermyx.pid")
	err = os.Remove(pidFile)
	if err != nil {
		engine.logger.Error(fmt.Sprintf("Failed to remove PID file: %v", err))
	}
	engine.logger.Info("PID file removed.")
	return err
}
