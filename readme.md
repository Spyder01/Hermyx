# 🌀 Hermyx

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.20+-blue)](https://golang.org/)
[![Build](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Status](https://img.shields.io/badge/status-beta-orange)]()

**Hermyx** is a blazing-fast, minimal reverse proxy with intelligent caching. Built on top of [`fasthttp`](https://github.com/valyala/fasthttp), it offers route-specific caching rules, graceful shutdown, flexible logging, and a clean YAML configuration — perfect for microservices, edge routing, or lightweight API gateways.

---

## 🚀 Features

* ⚡ **High Performance**: Powered by `fasthttp`, optimized for low-latency proxying.
* 🎯 **Per-Route Caching & Proxying**: Control cache behavior and target routing at the route level.
* 🧠 **Pluggable Caching Backends**: Choose between in-memory, disk-based, or Redis caching.
* ⏱ **TTL & Capacity Management**: Fine-grained control over cache expiry and size limits.
* 🗝️ **Custom Cache Keys**: Use `path`, `method`, `query`, and request `headers` to build smart cache keys.
* 🪵 **Flexible Logging**: Log to file and/or stdout with custom formats and prefixes.
* ✨ **Zero-Hassle YAML Config**: Simple, clean, and declarative.
* 🧹 **Graceful Shutdown**: Includes PID file management and safe cleanup.
* 🛠️ **Built-In Init Command**: Quickly scaffold a default config with `hermyx init`.

---

## ⚙️ Installation

> Coming soon as a prebuilt binary and via `go install`.

For now:

```bash
git clone https://github.com/your-username/hermyx.git
cd hermyx
go build -o hermyx ./cmd/go
```

---

## 📦 Usage

```bash
hermyx up --config ./configs/prod.yaml
hermyx down
hermyx init
```

---

## 📄 Configuration Overview

Hermyx is entirely configured via a single YAML file.

### Example

```yaml
log:
  toFile: true
  filePath: "./hermyx.log"
  toStdout: true
  prefix: "[Hermyx]"
  flags: 0
  debugEnabled: true

server:
  port: 8080

storage:
  path: "./.hermyx"

cache:
  enabled: true
  type: "redis"        # Options: "memory", "disk", "redis"
  ttl: 5m
  capacity: 1000
  maxContentSize: 1048576
  redis:
    address: "redis:6379"
    password: ""
    db: 0
    defaultTtl: 10s
    namespace: "hermyx:"
  keyConfig:
    type: ["path", "method", "query", "header"]
    headers:
      - key: "X-Request-User"
      - key: "X-Device-ID"
    excludeMethods: ["post", "put"]

routes:
  - name: "user-api"
    path: "^/api/users"
    target: "localhost:3000"
    include: [".*"]
    exclude: ["^/api/users/private"]
    cache:
      enabled: true
      ttl: 2m
      keyConfig:
        type: ["path", "query", "header"]
        headers:
          - key: "Authorization"
        excludeMethods: ["post"]
```

---

## 💡 Cache Types

Hermyx supports multiple caching backends. Choose one depending on your use case:

| Type     | Description                                        |
| -------- | -------------------------------------------------- |
| `memory` | In-memory LRU cache (fastest, non-persistent)      |
| `disk`   | Persistent file-based cache stored on disk         |
| `redis`  | Centralized cache with TTL support and namespacing |

---

## 🧾 Configuration Reference

### 🔹 `log`

| Field          | Type   | Description            |
| -------------- | ------ | ---------------------- |
| `toFile`       | bool   | Write logs to a file   |
| `filePath`     | string | Path to log file       |
| `toStdout`     | bool   | Write logs to stdout   |
| `prefix`       | string | Log line prefix        |
| `flags`        | int    | Go-style log flags     |
| `debugEnabled` | bool   | Print extra debug logs |

### 🔹 `server`

| Field  | Type | Description       |
| ------ | ---- | ----------------- |
| `port` | int  | Port to listen on |

### 🔹 `storage`

| Field  | Type   | Description                |
| ------ | ------ | -------------------------- |
| `path` | string | Path for storing PID, etc. |

### 🔹 `cache`

| Field            | Type        | Description                             |
| ---------------- | ----------- | --------------------------------------- |
| `enabled`        | bool        | Enable/disable global caching           |
| `type`           | string      | One of `memory`, `disk`, or `redis`     |
| `ttl`            | duration    | Global default TTL for cache entries    |
| `capacity`       | int         | Max cache entries (in memory/disk)      |
| `maxContentSize` | int         | Max body size (bytes) to store in cache |
| `keyConfig`      | KeyConfig   | Rules for generating cache keys         |
| `redis`          | RedisConfig | Redis-specific configuration            |

### 🔹 `routes`

| Field     | Type             | Description                              |
| --------- | ---------------- | ---------------------------------------- |
| `name`    | string           | Route identifier                         |
| `path`    | string           | Regex pattern for matching request paths |
| `target`  | string           | Upstream target address                  |
| `include` | \[]string        | List of sub-paths to include             |
| `exclude` | \[]string        | List of sub-paths to exclude             |
| `cache`   | CacheRouteConfig | Route-specific cache settings            |

### 🔹 `KeyConfig`

| Field            | Type            | Description                                                        |
| ---------------- | --------------- | ------------------------------------------------------------------ |
| `type`           | \[]string       | Which parts to include in key: `path`, `method`, `query`, `header` |
| `excludeMethods` | \[]string       | HTTP methods to ignore for caching (e.g. `POST`)                   |
| `headers`        | \[]HeaderConfig | Specific headers to include in the cache key                       |

### 🔹 `HeaderConfig`

| Field | Type   | Description            |
| ----- | ------ | ---------------------- |
| `key` | string | Header name to include |

---

## 🔁 How It Works

1. **Route Match**: Request is matched to a route using regex.
2. **Filter**: Include/exclude patterns are evaluated.
3. **Caching**:

   * Method or config can skip caching.
   * Cache key is built using selected components.
   * Cache is checked (in-memory, disk, or Redis).
4. **Proxy**:

   * If cache hit, serve response.
   * If miss, proxy request and cache result if allowed.
5. **Response**:

   * Adds `X-Hermyx-Cache: HIT` or `MISS` header.

---

## 🐞 Debugging Tips

* Enable `log.toStdout: true` and set `flags: 0` for clear log output.
* Inspect cache behavior using the `X-Hermyx-Cache` response header.
* For Redis, observe key TTL using:

```bash
redis-cli --ttl hermyx:<cache-key>
```

* Use meaningful request headers (like `X-User-ID` or `Authorization`) to build user-specific cache keys.

---

## 📌 Roadmap

* [ ] Add CLI auto-update
* [ ] Hot config reloading
* [ ] Built-in metrics via Prometheus
* [ ] Plugin system for auth/middleware

---

## 🧑‍💻 Contributing

PRs, bug reports, and ideas are welcome! Just fork and open a PR.

---

## 📄 License

MIT © [Suhan Bangera](https://github.com/suhanbangera)
