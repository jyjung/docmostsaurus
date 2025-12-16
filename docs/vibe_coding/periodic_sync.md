# 주기적 동기화 구현 가이드

`SYNC_INTERVAL` 환경변수를 사용하여 Docmost에서 주기적으로 데이터를 가져와 Docusaurus 포맷으로 변환하는 기능의 구현 가이드입니다.

## 현재 상태 분석

### 이미 구현된 것

| 항목 | 상태 | 위치 |
|------|------|------|
| Config에 SYNC_INTERVAL | ✅ 구현됨 | `internal/config/config.go` |
| Docmost API 클라이언트 | ✅ 구현됨 | `internal/docmost/client.go` |
| Docusaurus 변환 파이프라인 | ✅ 구현됨 | `cmd/docmost-file-sync/main.go` |
| Atomic Swap (Blue-Green) | ✅ 구현됨 | `cmd/docmost-file-sync/main.go` |
| 주기적 실행 로직 | ❌ 미구현 | 현재 1회 실행 후 종료 |

### 환경변수 설정 (`.env`)

```bash
DOCMOST_BASE_URL=http://192.168.31.101:3456
DOCMOST_EMAIL=your-email@example.com
DOCMOST_PASSWORD=your-password
OUTPUT_DIR=./output
SYNC_INTERVAL=1h
```

### 지원되는 SYNC_INTERVAL 형식

Go의 `time.ParseDuration` 형식을 따릅니다:

| 예시 | 의미 |
|------|------|
| `30s` | 30초 |
| `5m` | 5분 |
| `1h` | 1시간 |
| `1h30m` | 1시간 30분 |
| `24h` | 24시간 |

---

## 구현 방안

### 옵션 1: main.go에 스케줄러 내장 (권장)

`main.go`를 수정하여 `time.Ticker`를 사용한 주기적 실행 구현:

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "docmostsaurus/internal/config"
    "docmostsaurus/internal/docmost"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Context for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // 최초 1회 실행
    log.Println("Starting initial sync...")
    if err := runSync(ctx, cfg); err != nil {
        log.Printf("Initial sync failed: %v", err)
    }

    // 주기적 실행
    ticker := time.NewTicker(cfg.SyncInterval)
    defer ticker.Stop()

    log.Printf("Scheduler started. Next sync in %v", cfg.SyncInterval)

    for {
        select {
        case <-ticker.C:
            log.Println("Starting scheduled sync...")
            if err := runSync(ctx, cfg); err != nil {
                log.Printf("Scheduled sync failed: %v", err)
            }
            log.Printf("Next sync in %v", cfg.SyncInterval)

        case sig := <-sigChan:
            log.Printf("Received signal %v, shutting down...", sig)
            cancel()
            return
        }
    }
}

func runSync(ctx context.Context, cfg *config.Config) error {
    // 기존 동기화 로직
    client := docmost.NewClient(cfg.DocmostBaseURL, cfg.DocmostEmail, cfg.DocmostPassword)

    if err := client.Login(); err != nil {
        return err
    }

    spaces, err := client.ExportAllSpaces()
    if err != nil {
        return err
    }

    // 각 공간 처리 (후처리 파이프라인 + Atomic Swap)
    for _, space := range spaces {
        if err := processSpace(ctx, cfg, space); err != nil {
            log.Printf("Failed to process space %s: %v", space.Space.Name, err)
        }
    }

    return nil
}
```

---

## Graceful Shutdown 구현

프로세스 종료 시 진행 중인 작업을 안전하게 완료하는 메커니즘입니다.

### 구현 코드

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
)

type Scheduler struct {
    cfg       *config.Config
    ctx       context.Context
    cancel    context.CancelFunc
    wg        sync.WaitGroup
    isRunning bool
    mu        sync.Mutex
}

func NewScheduler(cfg *config.Config) *Scheduler {
    ctx, cancel := context.WithCancel(context.Background())
    return &Scheduler{
        cfg:    cfg,
        ctx:    ctx,
        cancel: cancel,
    }
}

func (s *Scheduler) Start() {
    // Signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        sig := <-sigChan
        log.Printf("Received signal: %v", sig)
        s.Shutdown()
    }()

    // Initial sync
    s.runSyncSafe()

    // Periodic sync
    ticker := time.NewTicker(s.cfg.SyncInterval)
    defer ticker.Stop()

    for {
        select {
        case <-s.ctx.Done():
            log.Println("Scheduler stopped")
            return
        case <-ticker.C:
            s.runSyncSafe()
        }
    }
}

func (s *Scheduler) runSyncSafe() {
    s.mu.Lock()
    if s.isRunning {
        s.mu.Unlock()
        log.Println("Sync already in progress, skipping...")
        return
    }
    s.isRunning = true
    s.mu.Unlock()

    s.wg.Add(1)
    defer func() {
        s.mu.Lock()
        s.isRunning = false
        s.mu.Unlock()
        s.wg.Done()
    }()

    if err := s.runSync(); err != nil {
        log.Printf("Sync failed: %v", err)
    }
}

func (s *Scheduler) Shutdown() {
    log.Println("Initiating graceful shutdown...")
    s.cancel()

    // Wait for running sync to complete (with timeout)
    done := make(chan struct{})
    go func() {
        s.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Println("Graceful shutdown completed")
    case <-time.After(30 * time.Second):
        log.Println("Shutdown timeout, forcing exit")
    }
}

func (s *Scheduler) runSync() error {
    // Context를 전달하여 중간에 취소 가능하게 함
    select {
    case <-s.ctx.Done():
        return s.ctx.Err()
    default:
    }

    // 동기화 로직...
    return nil
}
```

### Docker에서의 Graceful Shutdown

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /docmost-sync ./cmd/docmost-file-sync

FROM alpine:latest
COPY --from=builder /docmost-sync /docmost-sync

# SIGTERM을 제대로 받을 수 있도록 설정
STOPSIGNAL SIGTERM
ENTRYPOINT ["/docmost-sync"]
```

```yaml
# docker-compose.yml
services:
  docmost-sync:
    build: .
    stop_grace_period: 60s  # 종료 대기 시간
```

---

## 동시 실행 방지

### 방법 1: 프로세스 내 Mutex (권장)

위의 `Scheduler.runSyncSafe()` 코드에서 이미 구현됨:

```go
s.mu.Lock()
if s.isRunning {
    s.mu.Unlock()
    log.Println("Sync already in progress, skipping...")
    return
}
s.isRunning = true
s.mu.Unlock()
```

### 방법 2: 파일 기반 Lock

여러 프로세스가 동시에 실행될 수 있는 환경에서 사용:

```go
package lock

import (
    "fmt"
    "os"
    "syscall"
)

type FileLock struct {
    path string
    file *os.File
}

func NewFileLock(path string) *FileLock {
    return &FileLock{path: path}
}

func (l *FileLock) TryLock() error {
    file, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return err
    }

    err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
    if err != nil {
        file.Close()
        return fmt.Errorf("another instance is already running")
    }

    l.file = file

    // PID 기록
    file.Truncate(0)
    file.WriteString(fmt.Sprintf("%d\n", os.Getpid()))

    return nil
}

func (l *FileLock) Unlock() error {
    if l.file == nil {
        return nil
    }

    syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
    l.file.Close()
    os.Remove(l.path)

    return nil
}
```

사용 예:

```go
func main() {
    lock := lock.NewFileLock("/var/run/docmost-sync.lock")

    if err := lock.TryLock(); err != nil {
        log.Fatalf("Failed to acquire lock: %v", err)
    }
    defer lock.Unlock()

    // 동기화 로직...
}
```

---

## 로깅 구현

### 구조화된 로깅 (zerolog 사용)

```go
package main

import (
    "os"
    "time"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

func setupLogging() {
    // 콘솔 출력 (개발 환경)
    if os.Getenv("ENV") == "development" {
        log.Logger = log.Output(zerolog.ConsoleWriter{
            Out:        os.Stderr,
            TimeFormat: time.RFC3339,
        })
    } else {
        // JSON 출력 (프로덕션)
        zerolog.TimeFieldFormat = time.RFC3339
        log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
    }
}

func main() {
    setupLogging()

    log.Info().
        Str("interval", cfg.SyncInterval.String()).
        Str("output_dir", cfg.OutputDir).
        Msg("Starting docmost-sync")

    // 동기화 시작
    log.Info().
        Str("space", space.Name).
        Int("pages", space.TotalPages).
        Msg("Processing space")

    // 동기화 완료
    log.Info().
        Str("space", space.Name).
        Dur("duration", time.Since(startTime)).
        Msg("Space sync completed")

    // 에러 발생
    log.Error().
        Err(err).
        Str("space", space.Name).
        Msg("Failed to sync space")
}
```

### 로그 출력 예시

개발 환경 (ConsoleWriter):
```
2024-01-15T10:30:00+09:00 INF Starting docmost-sync interval=1h output_dir=./output
2024-01-15T10:30:01+09:00 INF Processing space space=Engineering pages=42
2024-01-15T10:30:15+09:00 INF Space sync completed space=Engineering duration=14.2s
```

프로덕션 (JSON):
```json
{"level":"info","time":"2024-01-15T10:30:00+09:00","interval":"1h","output_dir":"./output","message":"Starting docmost-sync"}
{"level":"info","time":"2024-01-15T10:30:01+09:00","space":"Engineering","pages":42,"message":"Processing space"}
{"level":"info","time":"2024-01-15T10:30:15+09:00","space":"Engineering","duration":14.2,"message":"Space sync completed"}
```

### 로그 레벨 설정

```bash
# .env
LOG_LEVEL=info  # debug, info, warn, error
```

```go
func setupLogging() {
    level := os.Getenv("LOG_LEVEL")
    switch level {
    case "debug":
        zerolog.SetGlobalLevel(zerolog.DebugLevel)
    case "warn":
        zerolog.SetGlobalLevel(zerolog.WarnLevel)
    case "error":
        zerolog.SetGlobalLevel(zerolog.ErrorLevel)
    default:
        zerolog.SetGlobalLevel(zerolog.InfoLevel)
    }
}
```

---

## 헬스체크 구현

### 방법 1: HTTP 헬스체크 엔드포인트

```go
package health

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"
)

type HealthChecker struct {
    mu            sync.RWMutex
    lastSyncTime  time.Time
    lastSyncError error
    syncCount     int64
    isRunning     bool
}

type HealthStatus struct {
    Status        string    `json:"status"`
    LastSync      time.Time `json:"last_sync,omitempty"`
    LastError     string    `json:"last_error,omitempty"`
    SyncCount     int64     `json:"sync_count"`
    IsRunning     bool      `json:"is_running"`
    Uptime        string    `json:"uptime"`
}

var (
    checker   = &HealthChecker{}
    startTime = time.Now()
)

func UpdateSyncStatus(err error) {
    checker.mu.Lock()
    defer checker.mu.Unlock()

    checker.lastSyncTime = time.Now()
    checker.lastSyncError = err
    checker.syncCount++
}

func SetRunning(running bool) {
    checker.mu.Lock()
    defer checker.mu.Unlock()
    checker.isRunning = running
}

func Handler(w http.ResponseWriter, r *http.Request) {
    checker.mu.RLock()
    defer checker.mu.RUnlock()

    status := HealthStatus{
        Status:    "healthy",
        LastSync:  checker.lastSyncTime,
        SyncCount: checker.syncCount,
        IsRunning: checker.isRunning,
        Uptime:    time.Since(startTime).String(),
    }

    if checker.lastSyncError != nil {
        status.Status = "degraded"
        status.LastError = checker.lastSyncError.Error()
    }

    // 마지막 동기화가 너무 오래됨
    if !checker.lastSyncTime.IsZero() {
        if time.Since(checker.lastSyncTime) > 2*time.Hour {
            status.Status = "unhealthy"
        }
    }

    w.Header().Set("Content-Type", "application/json")

    if status.Status == "unhealthy" {
        w.WriteHeader(http.StatusServiceUnavailable)
    }

    json.NewEncoder(w).Encode(status)
}

func StartServer(addr string) {
    http.HandleFunc("/health", Handler)
    http.HandleFunc("/healthz", Handler)  // Kubernetes 호환
    http.HandleFunc("/ready", Handler)    // Readiness probe

    go http.ListenAndServe(addr, nil)
}
```

main.go에서 사용:

```go
func main() {
    // 헬스체크 서버 시작
    health.StartServer(":8080")

    // 동기화 시작 시
    health.SetRunning(true)
    err := runSync()
    health.SetRunning(false)
    health.UpdateSyncStatus(err)
}
```

### 방법 2: 파일 기반 헬스체크

헬스체크 파일을 주기적으로 업데이트:

```go
func updateHealthFile(path string) error {
    data := map[string]interface{}{
        "timestamp":  time.Now().Format(time.RFC3339),
        "pid":        os.Getpid(),
        "status":     "running",
        "sync_count": syncCount,
    }

    content, _ := json.MarshalIndent(data, "", "  ")
    return os.WriteFile(path, content, 0644)
}
```

### Docker HEALTHCHECK

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /docmost-sync ./cmd/docmost-file-sync

FROM alpine:latest
RUN apk add --no-cache curl
COPY --from=builder /docmost-sync /docmost-sync

# 헬스체크 설정
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080
ENTRYPOINT ["/docmost-sync"]
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  docmost-sync:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DOCMOST_BASE_URL=${DOCMOST_BASE_URL}
      - DOCMOST_EMAIL=${DOCMOST_EMAIL}
      - DOCMOST_PASSWORD=${DOCMOST_PASSWORD}
      - SYNC_INTERVAL=1h
      - OUTPUT_DIR=/output
      - LOG_LEVEL=info
    volumes:
      - ./output:/output
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    restart: unless-stopped
    stop_grace_period: 60s
```

### Kubernetes Probes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: docmost-sync
spec:
  template:
    spec:
      containers:
      - name: sync
        image: docmostsaurus:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
```

---

## 전체 구현 체크리스트

- [ ] `SYNC_INTERVAL` 파싱 및 스케줄러 구현
- [ ] Graceful Shutdown (SIGTERM/SIGINT 처리)
- [ ] 동시 실행 방지 (Mutex 또는 파일 Lock)
- [ ] 구조화된 로깅 (zerolog)
- [ ] HTTP 헬스체크 엔드포인트
- [ ] Docker HEALTHCHECK 설정
- [ ] 에러 재시도 로직
- [ ] 메트릭스 수집 (선택사항)

---

## 참고 자료

- [Go time.ParseDuration](https://pkg.go.dev/time#ParseDuration)
- [zerolog - 구조화된 로깅](https://github.com/rs/zerolog)
- [Docker HEALTHCHECK](https://docs.docker.com/engine/reference/builder/#healthcheck)
- [Kubernetes Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
