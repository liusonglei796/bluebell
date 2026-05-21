# Design Specification: Observability Migration to Tempo, Mimir, and Pyroscope

## 1. Overview
The goal of this design is to modernize the community web application (Bluebell)'s telemetry and profiling stack by migrating from Jaeger (tracing) and Prometheus (pull-based metrics) to Grafana's modern cloud-native telemetry ecosystem:
- **Grafana Tempo**: Distributed tracing backend (replacing Jaeger).
- **Grafana Mimir**: High-performance, scalable, push-based Prometheus-compatible metrics backend (replacing standard Prometheus).
- **Grafana Pyroscope**: Continuous profiling backend to gain deep insights into CPU, memory allocation, and concurrency bottlenecks within the Go services.

All telemetry storage, query, and visualization backends will be integrated directly with Grafana.

---

## 2. System Architecture & Data Flow

```mermaid
graph TD
    subgraph Go App (Bluebell)
        App[Go Application]
        OTelSDK[OpenTelemetry SDK]
        PyroSDK[Pyroscope SDK]
    end

    subgraph Collection & Processing
        Collector[OpenTelemetry Collector]
    end

    subgraph Storage Backends (Grafana Observability Stack)
        Loki[Grafana Loki (Logs)]
        Tempo[Grafana Tempo (Traces)]
        Mimir[Grafana Mimir (Metrics)]
        Pyroscope[Grafana Pyroscope (Profiles)]
    end

    subgraph Visualization
        Grafana[Grafana Dashboards]
    end

    %% Data Flow
    App -->|Traces & Logs & Metrics| OTelSDK
    App -->|Push Profiles| Pyroscope
    
    OTelSDK -->|OTLP gRPC| Collector
    
    Collector -->|OTLP Traces| Tempo
    Collector -->|Prometheus Remote Write| Mimir
    Collector -->|OTLP HTTP Logs| Loki

    Tempo -->|Query (3200)| Grafana
    Mimir -->|Query (9009)| Grafana
    Loki -->|Query (3100)| Grafana
    Pyroscope -->|Query (4040)| Grafana
```

### Key Architectural Shifts:
1. **Push-based Metrics**: Instead of Prometheus scraping `/metrics` endpoints, OpenTelemetry SDK pushes metrics via OTLP to the OpenTelemetry Collector, which forwards them directly to Mimir via Prometheus Remote Write.
2. **Integrated Continuous Profiling**: The Go application continuously pushes lightweight CPU, memory, mutex, and block profiles directly to the Pyroscope monolithic service.
3. **Decoupled Telemetry**: The Go application remains agnostic to the specific tracing and metrics backend (tempo/mimir) since it interacts solely with the OpenTelemetry Collector using standard OTLP protocols.

---

## 3. Detailed Infrastructure Changes

### 3.1 Docker Compose Configurations
We will update `docker-compose.yml` and `docker-compose.dev.yml` to reflect the new services.

#### Grafana Tempo (Traces)
Tempo will run in single-binary monolithic mode with local block storage:
```yaml
  tempo:
    image: grafana/tempo:latest
    container_name: bluebell_tempo
    command: [ "-config.file=/etc/tempo/tempo.yaml" ]
    volumes:
      - ./tempo-config.yaml:/etc/tempo/tempo.yaml:ro
    ports:
      - "3200:3200"   # HTTP for Grafana Query
      - "4317"        # OTLP gRPC receiver
    networks:
      - bluebell_net
    restart: always
```

#### Grafana Mimir (Metrics)
Mimir will run in monolithic mode using a local filesystem backend:
```yaml
  mimir:
    image: grafana/mimir:latest
    container_name: bluebell_mimir
    command: [ "-config.file=/etc/mimir/mimir.yaml" ]
    volumes:
      - ./mimir-config.yaml:/etc/mimir/mimir.yaml:ro
    ports:
      - "9009:9009"   # HTTP for Grafana Query & Ingestion
    networks:
      - bluebell_net
    restart: always
```

#### Grafana Pyroscope (Profiling)
Pyroscope monolithic container:
```yaml
  pyroscope:
    image: grafana/pyroscope:latest
    container_name: bluebell_pyroscope
    ports:
      - "4040:4040"   # HTTP for Ingestion and Grafana Datasource
    networks:
      - bluebell_net
    restart: always
```

### 3.2 Monolithic Configurations to Add
We will create minimal, robust configuration files:
- `tempo-config.yaml`: Defines storage backend, block compression, and active receivers.
- `mimir-config.yaml`: Disables multi-tenancy for ease of development/single-instance production, sets local blocks directory, and configures inmemory ring stores.

### 3.3 OpenTelemetry Collector configuration (`otel-collector-config.yaml`)
- Modify `exporters` to route traces to Tempo:
  ```yaml
  exporters:
    otlp/tempo:
      endpoint: tempo:4317
      tls:
        insecure: true
  ```
- Replace the `prometheus` exporter with `prometheusremotewrite`:
  ```yaml
  exporters:
    prometheusremotewrite:
      endpoint: http://mimir:9009/api/v1/push
      tls:
        insecure: true
  ```
- Update `service.pipelines` to map metrics to `prometheusremotewrite` and traces to `otlp/tempo`.

---

## 4. Go Application Instrumentation

### 4.1 Dependency updates
Install the Pyroscope Go SDK:
```bash
go get github.com/grafana/pyroscope-go
```

### 4.2 Config schema extension
Modify `internal/config/config.go` to support Pyroscope configuration:
```go
type PyroscopeConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Endpoint    string `mapstructure:"endpoint"`
	ServiceName string `mapstructure:"service_name"`
}
```
Add `Pyroscope *PyroscopeConfig` to the main `Config` struct.

### 4.3 Profiler module
Create a clean, self-contained profiling initialization utility under `internal/infrastructure/profiler/pyroscope.go`:
```go
package profiler

import (
	"bluebell/internal/config"
	"github.com/grafana/pyroscope-go"
)

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

This profiler will be initialized in the main server (`cmd/bluebell/main.go`) as well as active workers/consumers if necessary.

---

## 5. Grafana Provisioning & Datasources

Update `grafana-provisioning/datasources/datasources.yaml`:
1. **Loki**: Point trace derivation `derivedFields` to the new `tempo` datasource UID.
2. **Prometheus / Mimir**: Update the existing `Prometheus` named datasource URL to Mimir's query URL: `http://mimir:9009/prometheus`.
3. **Tempo**: Create a new datasource of type `tempo` pointing to `http://tempo:3200`.
4. **Pyroscope**: Create a new datasource of type `pyroscope` pointing to `http://pyroscope:4040`.

---

## 6. Verification & Testing Plan

### Automated Build & Syntax Verification
- Ensure the Go application compiles successfully after importing the Pyroscope SDK.
- Validate the YAML files of docker-compose configs and collector configurations.

### Infrastructure Verification
- Run `docker compose up -d` to spin up the LGTM stack (Loki, Grafana, Tempo, Mimir, Pyroscope).
- Check standard health metrics for all storage services.

### Observability Check
- Send dummy HTTP/gRPC telemetry request to verify Mimir receives metrics.
- Generate application traffic (e.g. hitting HTTP endpoints) and verify in Grafana:
  - Metrics are scraped from Mimir.
  - Trace maps show up in Tempo.
  - CPU/Memory profile flames appear in Pyroscope.
