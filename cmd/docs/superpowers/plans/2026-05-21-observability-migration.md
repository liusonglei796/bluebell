# Observability Stack Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate community web application (Bluebell) tracing from Jaeger to Tempo, metrics from Prometheus to monolithic Mimir, and add continuous profiling via Pyroscope.

**Architecture:** Extend application configurations and instrument the Go application using the `pyroscope-go` SDK to push profiling streams to the Pyroscope server. Update the OpenTelemetry Collector configuration to route OTLP traces to Tempo and metrics to Mimir via Prometheus remote write, replacing the legacy jaeger and prometheus service containers in Docker Compose.

**Tech Stack:** Go 1.22+, Grafana Mimir, Grafana Tempo, Grafana Pyroscope, Grafana Loki, OpenTelemetry Collector, Docker Compose.

---

### Task 1: Go Application Dependency & Configuration Setup

**Files:**
- Modify: `D:\download\project\bluebell\internal\config\config.go:81-105`
- Modify: `D:\download\project\bluebell\config.yaml:55-59`
- Modify: `D:\download\project\bluebell\config.docker.toml:53-56`

- [ ] **Step 1: Install `pyroscope-go` SDK**
  Run: `go get github.com/grafana/pyroscope-go` in terminal.
  Expected: Success, additions to `go.mod` and `go.sum`.

- [ ] **Step 2: Add Pyroscope Config Struct to `config.go`**
  Modify: `D:\download\project\bluebell\internal\config\config.go`
  Target Content:
  ```go
  // OtelConfig OpenTelemetry 配置结构体
  type OtelConfig struct {
  	Enabled     bool   `mapstructure:"enabled"`
  	Endpoint    string `mapstructure:"endpoint"`
  	ServiceName string `mapstructure:"service_name"`
  }
  
  
  // Config 全局配置结构体
  // 使用指针类型以区分配置缺失和零值
  type Config struct {
  	App       *appConfig       `mapstructure:"app"`
  	Mysql     *mysqlConfig     `mapstructure:"mysql"`
  	Redis     *redisConfig     `mapstructure:"redis"`
  	Log       *logConfig       `mapstructure:"log"`
  	Snowflake *SnowflakeConfig `mapstructure:"snowflake"`
  	RateLimit *rateLimitConfig `mapstructure:"ratelimit"`
  	JWT       *jwtConfig       `mapstructure:"jwt"`
  	Timeout   *timeoutConfig   `mapstructure:"timeout"`
  	RabbitMQ  *rabbitmqConfig  `mapstructure:"rabbitmq"`
  	ES        *esConfig        `mapstructure:"es"`
  	Otel      *OtelConfig      `mapstructure:"otel"`
  	GitHub    *GitHubConfig    `mapstructure:"github"`
  }
  ```
  Replacement Content:
  ```go
  // OtelConfig OpenTelemetry 配置结构体
  type OtelConfig struct {
  	Enabled     bool   `mapstructure:"enabled"`
  	Endpoint    string `mapstructure:"endpoint"`
  	ServiceName string `mapstructure:"service_name"`
  }
  
  // PyroscopeConfig continuous profiling configuration
  type PyroscopeConfig struct {
  	Enabled     bool   `mapstructure:"enabled"`
  	Endpoint    string `mapstructure:"endpoint"`
  	ServiceName string `mapstructure:"service_name"`
  }
  
  // Config 全局配置结构体
  // 使用指针类型以区分配置缺失和零值
  type Config struct {
  	App       *appConfig       `mapstructure:"app"`
  	Mysql     *mysqlConfig     `mapstructure:"mysql"`
  	Redis     *redisConfig     `mapstructure:"redis"`
  	Log       *logConfig       `mapstructure:"log"`
  	Snowflake *SnowflakeConfig `mapstructure:"snowflake"`
  	RateLimit *rateLimitConfig `mapstructure:"ratelimit"`
  	JWT       *jwtConfig       `mapstructure:"jwt"`
  	Timeout   *timeoutConfig   `mapstructure:"timeout"`
  	RabbitMQ  *rabbitmqConfig  `mapstructure:"rabbitmq"`
  	ES        *esConfig        `mapstructure:"es"`
  	Otel      *OtelConfig      `mapstructure:"otel"`
  	Pyroscope *PyroscopeConfig `mapstructure:"pyroscope"`
  	GitHub    *GitHubConfig    `mapstructure:"github"`
  }
  ```

- [ ] **Step 3: Update `config.yaml` with Pyroscope section**
  Modify: `D:\download\project\bluebell\config.yaml`
  Target Content:
  ```yaml
  otel:
    enabled: true
    endpoint: "localhost:4317"
    service_name: "bluebell"
  ```
  Replacement Content:
  ```yaml
  otel:
    enabled: true
    endpoint: "localhost:4317"
    service_name: "bluebell"
  
  pyroscope:
    enabled: true
    endpoint: "http://localhost:4040"
    service_name: "bluebell"
  ```

- [ ] **Step 4: Update `config.docker.toml` with Pyroscope section**
  Modify: `D:\download\project\bluebell\config.docker.toml`
  Target Content:
  ```toml
  [otel]
  enabled = true
  endpoint = "otel-collector:4317"
  service_name = "bluebell"
  ```
  Replacement Content:
  ```toml
  [otel]
  enabled = true
  endpoint = "otel-collector:4317"
  service_name = "bluebell"
  
  [pyroscope]
  enabled = true
  endpoint = "http://pyroscope:4040"
  service_name = "bluebell"
  ```

- [ ] **Step 5: Commit**
  Run:
  ```bash
  git add internal/config/config.go config.yaml config.docker.toml
  git commit -m "chore: add pyroscope configuration structs and configurations"
  ```

---

### Task 2: Create Profiler Initializer and Integrate into Service Entrypoints

**Files:**
- Create: `D:\download\project\bluebell\internal\infrastructure\profiler\pyroscope.go`
- Modify: `D:\download\project\bluebell\cmd\bluebell\main.go:58-71`
- Modify: `D:\download\project\bluebell\cmd\consumer\sync\main.go:39-55`
- Modify: `D:\download\project\bluebell\cmd\consumer\vote\main.go:38-52`

- [ ] **Step 1: Implement `pyroscope.go` profiler utility**
  Create new file `D:\download\project\bluebell\internal\infrastructure\profiler\pyroscope.go`:
  ```go
  package profiler

  import (
  	"bluebell/internal/config"
  	"github.com/grafana/pyroscope-go"
  )

  // Init starts continuous profiling sessions using Grafana Pyroscope SDK
  func Init(cfg *config.PyroscopeConfig) (func() error, error) {
  	if cfg == nil || !cfg.Enabled {
  		return func() error { return nil }, nil
  	}

  	session, err := pyroscope.Start(pyroscope.Config{
  		ApplicationName: cfg.ServiceName,
  		ServerAddress:   cfg.Endpoint,
  		Logger:          pyroscope.StandardLogger,
  		ProfileTypes: []pyroscope.ProfileType{
  			pyroscope.ProfileCPU,
  			pyroscope.ProfileAllocObjects,
  			pyroscope.ProfileAllocSpace,
  			pyroscope.ProfileInuseObjects,
  			pyroscope.ProfileInuseSpace,
  			pyroscope.ProfileGoroutines,
  			pyroscope.ProfileMutexCount,
  			pyroscope.ProfileMutexDuration,
  			pyroscope.ProfileBlockCount,
  			pyroscope.ProfileBlockDuration,
  		},
  	})
  	if err != nil {
  		return nil, err
  	}

  	return session.Stop, nil
  }
  ```

- [ ] **Step 2: Add Profiler Initializer to HTTP Main Application Entrypoint**
  Modify: `D:\download\project\bluebell\cmd\bluebell\main.go`
  Target Content:
  ```go
  	// ====== 基础设施层：OpenTelemetry ======
  	// 初始化 OTel SDK（Traces + Metrics + Logs），必须在 Logger 之前
  	ctx := context.Background()
  	otelShutdown, err := bluebellotel.InitOTEL(ctx, cfg.Otel)
  	if err != nil {
  		fmt.Printf("init otel failed, err:%v\n", err)
  		return
  	}
  	defer func() {
  		if err := otelShutdown(ctx); err != nil {
  			fmt.Printf("otel shutdown error: %v\n", err)
  		}
  	}()
  ```
  Replacement Content:
  ```go
  	// ====== 基础设施层：OpenTelemetry ======
  	// 初始化 OTel SDK（Traces + Metrics + Logs），必须在 Logger 之前
  	ctx := context.Background()
  	otelShutdown, err := bluebellotel.InitOTEL(ctx, cfg.Otel)
  	if err != nil {
  		fmt.Printf("init otel failed, err:%v\n", err)
  		return
  	}
  	defer func() {
  		if err := otelShutdown(ctx); err != nil {
  			fmt.Printf("otel shutdown error: %v\n", err)
  		}
  	}()

  	// ====== 基础设施层：Pyroscope Profiler ======
  	if cfg.Pyroscope != nil {
  		pyroShutdown, err := di.InitPyroscope(cfg.Pyroscope)
  		if err != nil {
  			fmt.Printf("init pyroscope failed, err:%v\n", err)
  		} else {
  			defer pyroShutdown()
  		}
  	}
  ```
  Wait! Let's check `di.InitPyroscope` or if we should import `bluebell/internal/infrastructure/profiler` directly. Yes! Let's import the `profiler` package directly in `cmd/bluebell/main.go` and use it.
  Let's adjust this step to import `bluebell/internal/infrastructure/profiler` and call `profiler.Init`.
  Let's update the code block inside `cmd/bluebell/main.go` replacement to:
  ```go
  	// ====== 基础设施层：Pyroscope Profiler ======
  	if cfg.Pyroscope != nil {
  		pyroShutdown, err := profiler.Init(cfg.Pyroscope)
  		if err != nil {
  			fmt.Printf("init pyroscope failed, err:%v\n", err)
  		} else {
  			defer func() {
  				if err := pyroShutdown(); err != nil {
  					fmt.Printf("pyroscope shutdown error: %v\n", err)
  				}
  			}()
  		}
  	}
  ```
  And make sure `bluebell/internal/infrastructure/profiler` is imported in `main.go`.
  Import block target:
  ```go
  	bluebellotel "bluebell/internal/infrastructure/otel"
  	database "bluebell/internal/infrastructure/persistence/mysql"
  ```
  Import block replacement:
  ```go
  	bluebellotel "bluebell/internal/infrastructure/otel"
  	"bluebell/internal/infrastructure/profiler"
  	database "bluebell/internal/infrastructure/persistence/mysql"
  ```

- [ ] **Step 3: Add Profiler to sync consumer**
  Modify: `D:\download\project\bluebell\cmd\consumer\sync\main.go`
  First, let's view this file around lines 30-60 to get context. We will check it when executing, but let's formulate the exact modification:
  Include `bluebell/internal/infrastructure/profiler` in imports.
  After `InitOTEL(otelCtx, cfg.Otel)` setup, add:
  ```go
  	// ====== 基础设施层：Pyroscope Profiler ======
  	if cfg.Pyroscope != nil {
  		pyroShutdown, err := profiler.Init(cfg.Pyroscope)
  		if err != nil {
  			fmt.Printf("init pyroscope failed, err:%v\n", err)
  		} else {
  			defer func() {
  				if err := pyroShutdown(); err != nil {
  					fmt.Printf("pyroscope shutdown error: %v\n", err)
  				}
  			}()
  		}
  	}
  ```

- [ ] **Step 4: Add Profiler to vote consumer**
  Modify: `D:\download\project\bluebell\cmd\consumer\vote\main.go`
  Include `bluebell/internal/infrastructure/profiler` in imports.
  After `InitOTEL(ctx, cfg.Otel)` setup, add:
  ```go
  	// ====== 基础设施层：Pyroscope Profiler ======
  	if cfg.Pyroscope != nil {
  		pyroShutdown, err := profiler.Init(cfg.Pyroscope)
  		if err != nil {
  			fmt.Printf("init pyroscope failed, err:%v\n", err)
  		} else {
  			defer func() {
  				if err := pyroShutdown(); err != nil {
  					fmt.Printf("pyroscope shutdown error: %v\n", err)
  				}
  			}()
  		}
  	}
  ```

- [ ] **Step 5: Commit**
  Run:
  ```bash
  git add internal/infrastructure/profiler/pyroscope.go cmd/bluebell/main.go cmd/consumer/sync/main.go cmd/consumer/vote/main.go
  git commit -m "feat: integrate continuous profiling via Pyroscope SDK in app and consumers"
  ```

---

### Task 3: Provision Monolithic Observability Config Files

**Files:**
- Create: `D:\download\project\bluebell\tempo-config.yaml`
- Create: `D:\download\project\bluebell\mimir-config.yaml`

- [ ] **Step 1: Write `tempo-config.yaml`**
  Create file `D:\download\project\bluebell\tempo-config.yaml` with the following monolithic config:
  ```yaml
  server:
    http_listen_port: 3200

  distributor:
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318

  ingester:
    lifecycler:
      ring:
        kvstore:
          store: inmemory
        replication_factor: 1

  compactor:
    ring:
      kvstore:
        store: inmemory

  storage:
    trace:
      backend: local
      local:
        path: /var/tempo/blocks
  ```

- [ ] **Step 2: Write `mimir-config.yaml`**
  Create file `D:\download\project\bluebell\mimir-config.yaml` with the following monolithic config:
  ```yaml
  target: all
  multitenancy_enabled: false

  server:
    http_listen_port: 9009
    grpc_listen_port: 9095

  common:
    path_prefix: /var/mimir
    storage:
      backend: local
      local:
        dir: /var/mimir/data
    replication_factor: 1
    ring:
      kvstore:
        store: inmemory

  blocks_storage:
    backend: local
    local:
      directory: /var/mimir/blocks
    bucket_store:
      sync_dir: /var/mimir/tsdb-sync

  compactor:
    data_dir: /var/mimir/compact
    ring:
      kvstore:
        store: inmemory

  ingester:
    ring:
      kvstore:
        store: inmemory
  ```

- [ ] **Step 3: Commit**
  Run:
  ```bash
  git add tempo-config.yaml mimir-config.yaml
  git commit -m "chore: add tempo and mimir monolithic config files"
  ```

---

### Task 4: Configure OpenTelemetry Collector to route to Tempo and Mimir

**Files:**
- Modify: `D:\download\project\bluebell\otel-collector-config.yaml`

- [ ] **Step 1: Update OpenTelemetry Collector Exporters and Pipelines**
  Modify: `D:\download\project\bluebell\otel-collector-config.yaml`
  Target Content:
  ```yaml
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
  
  processors:
    batch:
  
  exporters:
    otlp:
      endpoint: jaeger:4317
      tls:
        insecure: true
  
    otlphttp/loki:
      endpoint: http://loki:3100/otlp
      tls:
        insecure: true
  
    prometheus:
      endpoint: 0.0.0.0:8889
  
    debug:
  
  service:
    pipelines:
      traces:
        receivers: [otlp]
        processors: [batch]
        exporters: [otlp, debug]
      metrics:
        receivers: [otlp]
        processors: [batch]
        exporters: [prometheus, debug]
      logs:
        receivers: [otlp]
        processors: [batch]
        exporters: [otlphttp/loki, debug]
  ```
  Replacement Content:
  ```yaml
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
  
  processors:
    batch:
  
  exporters:
    otlp/tempo:
      endpoint: tempo:4317
      tls:
        insecure: true
  
    prometheusremotewrite:
      endpoint: http://mimir:9009/api/v1/push
      tls:
        insecure: true
  
    otlphttp/loki:
      endpoint: http://loki:3100/otlp
      tls:
        insecure: true
  
    debug:
  
  service:
    pipelines:
      traces:
        receivers: [otlp]
        processors: [batch]
        exporters: [otlp/tempo, debug]
      metrics:
        receivers: [otlp]
        processors: [batch]
        exporters: [prometheusremotewrite, debug]
      logs:
        receivers: [otlp]
        processors: [batch]
        exporters: [otlphttp/loki, debug]
  ```

- [ ] **Step 2: Commit**
  Run:
  ```bash
  git add otel-collector-config.yaml
  git commit -m "chore: configure otel-collector to export traces to Tempo and remote write metrics to Mimir"
  ```

---

### Task 5: Update Docker Compose Infrastructure

**Files:**
- Modify: `D:\download\project\bluebell\docker-compose.yml:124-207`
- Modify: `D:\download\project\bluebell\docker-compose.dev.yml:83-164`

- [ ] **Step 1: Update production `docker-compose.yml`**
  Modify: `D:\download\project\bluebell\docker-compose.yml`
  Target Content:
  ```yaml
    # ====== 可观测性基础设施 ======
  
    # OpenTelemetry Collector — 统一收集入口
    otel-collector:
      image: otel/opentelemetry-collector-contrib:latest
      container_name: bluebell_otel_collector
      command: ["--config=/etc/otel-collector-config.yaml"]
      volumes:
        - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml:ro
      ports:
        - "4317:4317"   # OTLP gRPC
        - "4318:4318"   # OTLP HTTP
        - "8889:8889"   # Prometheus exporter
      depends_on:
        - jaeger
        - loki
      networks:
        - bluebell_net
      restart: always
  
    # Jaeger — 分布式追踪后端 (All-in-one)
    jaeger:
      image: jaegertracing/all-in-one:latest
      container_name: bluebell_jaeger
      ports:
        - "16686:16686" # UI
        - "4317"       # OTLP gRPC
      networks:
        - bluebell_net
      restart: always
  
    # Loki — 日志聚合后端
    loki:
      image: grafana/loki:latest
      container_name: bluebell_loki
      command: ["-config.file=/etc/loki/local-config.yaml"]
      volumes:
        - ./loki-config.yaml:/etc/loki/local-config.yaml:ro
      ports:
        - "3100:3100"
      networks:
        - bluebell_net
      restart: always
  
    # Prometheus — 指标后端（从 OTel Collector 拉取）
    prometheus:
      image: prom/prometheus:latest
      container_name: bluebell_prometheus
      volumes:
        - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
        - ./prometheus-alerts.yml:/etc/prometheus/prometheus-alerts.yml:ro
      ports:
        - "9090:9090"
      depends_on:
        - otel-collector
      networks:
        - bluebell_net
      restart: always
  
    # Grafana — 统一可视化仪表盘
    grafana:
      image: grafana/grafana:latest
      container_name: bluebell_grafana
      environment:
        - GF_AUTH_ANONYMOUS_ENABLED=true
        - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
      volumes:
        - ./grafana-provisioning/datasources:/etc/grafana/provisioning/datasources:ro
        - ./grafana-provisioning/dashboards:/etc/grafana/provisioning/dashboards:ro
        - ./grafana-dashboards:/etc/grafboards:ro
      ports:
        - "3000:3000"
      depends_on:
        - jaeger
        - loki
        - prometheus
      networks:
        - bluebell_net
      restart: always
  ```
  *(Note: Line 193 in the original file actually had `- ./grafana-dashboards:/etc/grafana/dashboards:ro`, we make sure it's correct)*
  Replacement Content:
  ```yaml
    # ====== 可观测性基础设施 ======
  
    # OpenTelemetry Collector — 统一收集入口
    otel-collector:
      image: otel/opentelemetry-collector-contrib:latest
      container_name: bluebell_otel_collector
      command: ["--config=/etc/otel-collector-config.yaml"]
      volumes:
        - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml:ro
      ports:
        - "4317:4317"   # OTLP gRPC
        - "4318:4318"   # OTLP HTTP
      depends_on:
        - tempo
        - loki
        - mimir
      networks:
        - bluebell_net
      restart: always
  
    # Tempo — 分布式追踪后端
    tempo:
      image: grafana/tempo:latest
      container_name: bluebell_tempo
      command: ["-config.file=/etc/tempo/tempo-config.yaml"]
      volumes:
        - ./tempo-config.yaml:/etc/tempo/tempo-config.yaml:ro
      ports:
        - "3200:3200"   # UI/API
      networks:
        - bluebell_net
      restart: always
  
    # Loki — 日志聚合后端
    loki:
      image: grafana/loki:latest
      container_name: bluebell_loki
      command: ["-config.file=/etc/loki/local-config.yaml"]
      volumes:
        - ./loki-config.yaml:/etc/loki/local-config.yaml:ro
      ports:
        - "3100:3100"
      networks:
        - bluebell_net
      restart: always
  
    # Mimir — 高可用、多租户指标后端
    mimir:
      image: grafana/mimir:latest
      container_name: bluebell_mimir
      command: ["-config.file=/etc/mimir/mimir-config.yaml"]
      volumes:
        - ./mimir-config.yaml:/etc/mimir/mimir-config.yaml:ro
      ports:
        - "9009:9009"
      networks:
        - bluebell_net
      restart: always
  
    # Pyroscope — 持续性能剖析后端
    pyroscope:
      image: grafana/pyroscope:latest
      container_name: bluebell_pyroscope
      ports:
        - "4040:4040"
      networks:
        - bluebell_net
      restart: always
  
    # Grafana — 统一可视化仪表盘
    grafana:
      image: grafana/grafana:latest
      container_name: bluebell_grafana
      environment:
        - GF_AUTH_ANONYMOUS_ENABLED=true
        - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
      volumes:
        - ./grafana-provisioning/datasources:/etc/grafana/provisioning/datasources:ro
        - ./grafana-provisioning/dashboards:/etc/grafana/provisioning/dashboards:ro
        - ./grafana-dashboards:/etc/grafana/dashboards:ro
      ports:
        - "3000:3000"
      depends_on:
        - tempo
        - loki
        - mimir
        - pyroscope
      networks:
        - bluebell_net
      restart: always
  ```

- [ ] **Step 2: Update development `docker-compose.dev.yml`**
  Modify: `D:\download\project\bluebell\docker-compose.dev.yml`
  Target Content:
  ```yaml
    # ====== 可观测性基础设施 ======
  
    # OpenTelemetry Collector — 统一收集入口
    otel-collector:
      image: otel/opentelemetry-collector-contrib:latest
      container_name: bluebell_otel_collector
      command: ["--config=/etc/otel-collector-config.yaml"]
      volumes:
        - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml:ro
      ports:
        - "4317:4317"   # OTLP gRPC
        - "4318:4318"   # OTLP HTTP
        - "8889:8889"   # Prometheus exporter
      depends_on:
        - jaeger
        - loki
      networks:
        - bluebell_net
      restart: always
  
    # Jaeger — 分布式追踪后端 (All-in-one)
    jaeger:
      image: jaegertracing/all-in-one:latest
      container_name: bluebell_jaeger
      ports:
        - "16686:16686" # UI
        - "4317"       # OTLP gRPC
      networks:
        - bluebell_net
      restart: always
  
    # Loki — 日志聚合后端
    loki:
      image: grafana/loki:latest
      container_name: bluebell_loki
      command: ["-config.file=/etc/loki/local-config.yaml"]
      volumes:
        - ./loki-config.yaml:/etc/loki/local-config.yaml:ro
      ports:
        - "3100:3100"
      networks:
        - bluebell_net
      restart: always
  
    # Prometheus — 指标后端（从 OTel Collector 拉取）
    prometheus:
      image: prom/prometheus:latest
      container_name: bluebell_prometheus
      volumes:
        - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      ports:
        - "9090:9090"
      depends_on:
        - otel-collector
      networks:
        - bluebell_net
      restart: always
  
    # Grafana — 统一可视化仪表盘
    grafana:
      image: grafana/grafana:latest
      container_name: bluebell_grafana
      environment:
        - GF_AUTH_ANONYMOUS_ENABLED=true
        - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
      volumes:
        - ./grafana-provisioning/datasources:/etc/grafana/provisioning/datasources:ro
        - ./grafana-provisioning/dashboards:/etc/grafana/provisioning/dashboards:ro
        - ./grafana-dashboards:/etc/grafana/dashboards:ro
      ports:
        - "3000:3000"
      depends_on:
        - jaeger
        - loki
        - prometheus
      networks:
        - bluebell_net
      restart: always
  ```
  Replacement Content:
  ```yaml
    # ====== 可观测性基础设施 ======
  
    # OpenTelemetry Collector — 统一收集入口
    otel-collector:
      image: otel/opentelemetry-collector-contrib:latest
      container_name: bluebell_otel_collector
      command: ["--config=/etc/otel-collector-config.yaml"]
      volumes:
        - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml:ro
      ports:
        - "4317:4317"   # OTLP gRPC
        - "4318:4318"   # OTLP HTTP
      depends_on:
        - tempo
        - loki
        - mimir
      networks:
        - bluebell_net
      restart: always
  
    # Tempo — 分布式追踪后端
    tempo:
      image: grafana/tempo:latest
      container_name: bluebell_tempo
      command: ["-config.file=/etc/tempo/tempo-config.yaml"]
      volumes:
        - ./tempo-config.yaml:/etc/tempo/tempo-config.yaml:ro
      ports:
        - "3200:3200"   # UI/API
      networks:
        - bluebell_net
      restart: always
  
    # Loki — 日志聚合后端
    loki:
      image: grafana/loki:latest
      container_name: bluebell_loki
      command: ["-config.file=/etc/loki/local-config.yaml"]
      volumes:
        - ./loki-config.yaml:/etc/loki/local-config.yaml:ro
      ports:
        - "3100:3100"
      networks:
        - bluebell_net
      restart: always
  
    # Mimir — 高可用、多租户指标后端
    mimir:
      image: grafana/mimir:latest
      container_name: bluebell_mimir
      command: ["-config.file=/etc/mimir/mimir-config.yaml"]
      volumes:
        - ./mimir-config.yaml:/etc/mimir/mimir-config.yaml:ro
      ports:
        - "9009:9009"
      networks:
        - bluebell_net
      restart: always
  
    # Pyroscope — 持续性能剖析后端
    pyroscope:
      image: grafana/pyroscope:latest
      container_name: bluebell_pyroscope
      ports:
        - "4040:4040"
      networks:
        - bluebell_net
      restart: always
  
    # Grafana — 统一可视化仪表盘
    grafana:
      image: grafana/grafana:latest
      container_name: bluebell_grafana
      environment:
        - GF_AUTH_ANONYMOUS_ENABLED=true
        - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
      volumes:
        - ./grafana-provisioning/datasources:/etc/grafana/provisioning/datasources:ro
        - ./grafana-provisioning/dashboards:/etc/grafana/provisioning/dashboards:ro
        - ./grafana-dashboards:/etc/grafana/dashboards:ro
      ports:
        - "3000:3000"
      depends_on:
        - tempo
        - loki
        - mimir
        - pyroscope
      networks:
        - bluebell_net
      restart: always
  ```

- [ ] **Step 3: Commit**
  Run:
  ```bash
  git add docker-compose.yml docker-compose.dev.yml
  git commit -m "chore: replace jaeger/prometheus with tempo/mimir and add pyroscope to docker-compose"
  ```

---

### Task 6: Update Grafana Provisioning Datasources

**Files:**
- Modify: `D:\download\project\bluebell\grafana-provisioning\datasources\datasources.yaml`

- [ ] **Step 1: Modify `datasources.yaml`**
  Modify: `D:\download\project\bluebell\grafana-provisioning\datasources\datasources.yaml`
  Target Content:
  ```yaml
  apiVersion: 1
  
  datasources:
    - name: Prometheus
      type: prometheus
      access: proxy
      url: http://prometheus:9090
  
    - name: Loki
      type: loki
      access: proxy
      url: http://loki:3100
      jsonData:
        derivedFields:
          - datasourceUid: jaeger
            matcherRegex: "trace_id\":\"(\\w+)\""
            name: TraceID
            url: "$${__value.raw}"
  
    - name: Jaeger
      type: jaeger
      uid: jaeger
      access: proxy
      url: http://jaeger:16686
  ```
  Replacement Content:
  ```yaml
  apiVersion: 1
  
  datasources:
    - name: Prometheus
      type: prometheus
      access: proxy
      url: http://mimir:9009/prometheus
  
    - name: Loki
      type: loki
      access: proxy
      url: http://loki:3100
      jsonData:
        derivedFields:
          - datasourceUid: tempo
            matcherRegex: "trace_id\":\"(\\w+)\""
            name: TraceID
            url: "$${__value.raw}"
  
    - name: Tempo
      type: tempo
      uid: tempo
      access: proxy
      url: http://tempo:3200
  
    - name: Pyroscope
      type: pyroscope
      uid: pyroscope
      access: proxy
      url: http://pyroscope:4040
  ```

- [ ] **Step 2: Commit**
  Run:
  ```bash
  git add grafana-provisioning/datasources/datasources.yaml
  git commit -m "chore: update Grafana provisioning datasources to include Tempo, Mimir, and Pyroscope"
  ```

---

### Task 7: Verification and End-to-End Testing

**Files:**
- None

- [ ] **Step 1: Test Compile Go Application**
  Run: `go build -o tmp/bluebell cmd/bluebell/main.go`
  Expected: Success, binary compiled with no compiler errors.

- [ ] **Step 2: Start Telemetry Stack Containers**
  Run: `docker compose -f docker-compose.dev.yml up -d mimir tempo loki pyroscope otel-collector grafana`
  Expected: Containers successfully pulled/started and running without immediate crashes. Check status using `docker compose -f docker-compose.dev.yml ps`.

- [ ] **Step 3: Run the Application Locally**
  Run: `go run cmd/bluebell/main.go -conf ./config.yaml`
  Expected: Application starts successfully, initializes OTel, profiling, and logs. It starts HTTP server listening on port 8080.

- [ ] **Step 4: Verify Dashboard Visualization**
  Instruct user to open Grafana (http://localhost:3000):
  - Go to Explore page.
  - Select Prometheus datasource and run queries to verify metrics scrape is succeeding through Mimir.
  - Select Pyroscope datasource and check CPU/memory profiling flamegraphs.
  - Select Tempo datasource and verify traces are visible.
