package engine

import (
	"fmt"
	"hermyx/pkg/cache"
	"hermyx/pkg/cachemanager"
	"hermyx/pkg/middleware"
	"hermyx/pkg/models"
	"hermyx/pkg/utils/fs"
	"hermyx/pkg/utils/hash"
	"hermyx/pkg/utils/logger"
	"hermyx/pkg/utils/system"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"
)

type compiledRoute struct {
	Route        *models.RouteConfig
	PathPattern  *regexp.Regexp
	IncludeRegex *regexp.Regexp
	ExcludeRegex *regexp.Regexp
}

type HermyxEngine struct {
	config          *models.HermyxConfig
	logger          *logger.Logger
	cacheManager    *cachemanager.CacheManager
	compiledRoutes  []compiledRoute
	configPath      string
	pid             uint64
	hostClients     map[string]*fasthttp.HostClient
	MiddlewareChain *middleware.Chain
}

func InstantiateHermyxEngine(configPath string) *HermyxEngine {
	var config models.HermyxConfig

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Unable to read the config-path %s: %v", configPath, err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Unable to parse the config at %s: %v", configPath, err)
	}

	logger_, err := logger.NewLogger(config.Log)
	if err != nil {
		log.Fatalf("Unable to instantiate the logger: %v", err)
	}

	// Intelligent defaults
	if config.Server == nil || config.Server.Port == 0 {
		logger_.Warn("Server port not specified defaulting to random port.")
		port, err := system.GetFreePort()
		if err != nil {
			log.Fatalf("Unable to assign a free port.")
		}

		logger_.Info(fmt.Sprintf("Assigned port %d", port))
		config.Server = &models.ServerConfig{
			Port: uint16(port),
		}
	}
	if config.Cache.Capacity == 0 {
		logger_.Warn(fmt.Sprintf("Global cache capacity not specified, assigning the value of %d items", 1000))
		config.Cache.Capacity = 1000
	}
	if config.Cache.Ttl == 0 {
		logger_.Warn(fmt.Sprintf("Global cache time-to-live specified, assigning the value of %d", 300000000000))
		config.Cache.Ttl = 300000000000
	}
	if config.Cache.MaxContentSize == 0 {
		logger_.Warn(fmt.Sprintf("Global cache max content size not specified, assigning the value of %d bytes", 1*1024*1024))
		config.Cache.MaxContentSize = 1 * 1024 * 1024
	}
	if config.Cache.Type == "" {
		logger_.Warn(fmt.Sprintf("Global cache type not specified, assigning the %s cache", models.CACHE_TYPE_MEMORY))
		config.Cache.Type = models.CACHE_TYPE_MEMORY
	}

	if config.Storage == nil || config.Storage.Path == "" {
		logger_.Warn("Storage path not specified assigning the default path...")
		programDataDir, err := fs.GetUserAppDataDir("hermyx")
		if err != nil {
			log.Fatalf("Unable to identify program storage path due to %v", err)
		}

		absConfigPath, err := filepath.Abs(configPath)
		if err != nil {
			log.Fatalf("Unable to identify program storage path due to %v", err)
		}

		storageDir := filepath.Join(programDataDir, hash.HashString(absConfigPath))
		logger_.Info(fmt.Sprintf("Assigning storage path as %s", storageDir))

		config.Storage = &models.StorageConfig{storageDir}
	}
	if config.Routes == nil {
		config.Routes = []models.RouteConfig{}
	}
	if config.Log == nil {
		config.Log = &models.LogConfig{
			ToStdout: true,
			Prefix:   "[Hermyx]",
			Flags:    0,
		}
	}

	var cache_ cachemanager.ICache

	switch config.Cache.Type {
	case models.CACHE_TYPE_MEMORY:
		cache_ = cache.NewCache(config.Cache.Capacity)

	case models.CACHE_TYPE_DISK:
		diskCache, err := cache.NewDiskCache(config.Storage.Path, config.Cache.Capacity)
		if err != nil {
			log.Fatalf("Unable to instantiate the disk-cache: %v", err)
		}
		cache_ = diskCache
	case models.CACHE_TYPE_REDIS:
		if config.Cache.Redis == nil {
			log.Fatalf("Redis config hasn't been provided.")
		}

		cache_ = cache.NewRedisCache(config.Cache.Redis)

	}

	cacheManager := cachemanager.NewCacheManager(cache_)

	engine := &HermyxEngine{
		config:       &config,
		logger:       logger_,
		cacheManager: cacheManager,
		configPath:   configPath,
		pid:          uint64(os.Getpid()),
		hostClients:  make(map[string]*fasthttp.HostClient),
	}

	engine.compileRoutes()

	return engine
}

func (engine *HermyxEngine) getClientForTarget(target string) *fasthttp.HostClient {
	addr := strings.TrimPrefix(strings.TrimPrefix(target, "http://"), "https://")

	if client, ok := engine.hostClients[addr]; ok {
		return client
	}

	client := &fasthttp.HostClient{
		Addr:     addr,
		MaxConns: 10000, // Tune this
	}
	engine.hostClients[addr] = client
	return client
}
