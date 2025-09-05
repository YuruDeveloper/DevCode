# DevCode 이벤트 시스템 실패 처리 방안

## 현재 상황 분석

### 기존 실패 처리 메커니즘
DevCode는 `src/events/typedBus.go:43-47`에서 기본적인 패닉 처리를 구현하고 있습니다:

```go
defer func() {
    if recover := recover(); recover != nil {
        fmt.Printf("PANIC %v\n %s\n",recover,debug.Stack())
    }
}()
```

**현재 방식의 문제점:**
- 패닉 발생 시 단순히 로그만 출력하고 이벤트 유실
- 실패한 이벤트에 대한 재처리 메커니즘 없음
- 실패 통계나 모니터링 불가
- 실패 원인 분석 어려움

---

## 실패 시나리오 분석

### 1. 이벤트 핸들러 레벨 실패

**발생 상황:**
```go
// 서비스에서 이벤트 처리 중 패닉
instance.bus.ToolCallEvent.Subscribe(constants.ToolService, func(event events.Event[dto.ToolCallData]) {
    // nil 포인터 접근, 타입 변환 실패 등
    instance.ProcessToolCall(event.Data) // <- 여기서 패닉
})
```

**영향:**
- 해당 이벤트만 유실, 다른 구독자에게는 전달됨
- 고루틴 종료로 인한 메모리 누수는 없음 (ants pool 관리)

### 2. 이벤트 발행 레벨 실패

**발생 상황:**
```go
// 이벤트 발행 시 고루틴 풀 문제
instance.pool.Submit(func() { ... }) // <- 풀 포화, 메모리 부족 등
```

**영향:**
- 모든 구독자가 이벤트를 받지 못함
- 시스템 전체 기능 마비 가능성

### 3. 구독자 등록/해제 시 실패

**발생 상황:**
```go
// 동시성 문제로 인한 맵 손상
instance.handlers[source] = handler // <- 경합 상태
```

**영향:**
- 이벤트 버스 전체 불안정
- 예측 불가능한 동작

---

## 개선 방안

### 1. 강화된 실패 처리 시스템

#### A. 실패 이벤트 추가
```go
// src/dto/dto.go에 추가
type SystemFailureData struct {
    FailedEvent    string          `json:"failed_event"`
    Source         constants.Source `json:"source"`
    Error          string          `json:"error"`
    StackTrace     string          `json:"stack_trace"`
    RetryCount     int             `json:"retry_count"`
    Timestamp      time.Time       `json:"timestamp"`
}
```

#### B. Dead Letter Queue 구현
```go
// src/events/deadLetterQueue.go
type DeadLetterQueue struct {
    failedEvents chan SystemFailureData
    maxRetries   int
    retryDelay   time.Duration
    storage      FailureStorage
}

func (dlq *DeadLetterQueue) HandleFailedEvent(eventType string, source constants.Source, err error, stackTrace string) {
    failure := SystemFailureData{
        FailedEvent: eventType,
        Source:      source,
        Error:       err.Error(),
        StackTrace:  stackTrace,
        RetryCount:  0,
        Timestamp:   time.Now(),
    }
    
    select {
    case dlq.failedEvents <- failure:
    default:
        // DLQ도 가득 찬 경우 파일에 기록
        dlq.storage.PersistFailure(failure)
    }
}
```

### 2. 개선된 TypedBus 구현

```go
// src/events/enhancedTypedBus.go
type EnhancedTypedBus[T any] struct {
    handlers         map[constants.Source]func(Event[T])
    pool            *ants.Pool
    handlerMutex    sync.RWMutex
    deadLetterQueue *DeadLetterQueue
    eventType       string
    maxRetries      int
    circuitBreaker  *CircuitBreaker
}

func (instance *EnhancedTypedBus[T]) Publish(event Event[T]) error {
    // Circuit Breaker 상태 확인
    if !instance.circuitBreaker.CanExecute() {
        return fmt.Errorf("circuit breaker is open for %s events", instance.eventType)
    }
    
    instance.handlerMutex.RLock()
    handlers := make([]func(Event[T]), 0, len(instance.handlers))
    for _, handler := range instance.handlers {
        handlers = append(handlers, handler)
    }
    instance.handlerMutex.RUnlock()
    
    successCount := 0
    totalCount := len(handlers)
    
    for source, handler := range instance.handlers {
        err := instance.pool.Submit(func() {
            defer func() {
                if r := recover(); r != nil {
                    // 실패 정보를 DLQ에 전송
                    instance.deadLetterQueue.HandleFailedEvent(
                        instance.eventType,
                        source,
                        fmt.Errorf("panic: %v", r),
                        string(debug.Stack()),
                    )
                    instance.circuitBreaker.RecordFailure()
                }
            }()
            
            handler(event)
            instance.circuitBreaker.RecordSuccess()
            atomic.AddInt32(&successCount, 1)
        })
        
        if err != nil {
            // 고루틴 풀 제출 실패
            instance.deadLetterQueue.HandleFailedEvent(
                instance.eventType,
                source,
                err,
                "goroutine pool submission failed",
            )
        }
    }
    
    // 부분적 실패 체크
    if successCount < totalCount {
        return fmt.Errorf("partial failure: %d/%d handlers succeeded", successCount, totalCount)
    }
    
    return nil
}
```

### 3. Circuit Breaker 패턴

```go
// src/events/circuitBreaker.go
type CircuitBreaker struct {
    maxFailures     int
    resetTimeout    time.Duration
    failureCount    int64
    lastFailureTime time.Time
    state          CircuitState
    mutex          sync.RWMutex
}

type CircuitState int

const (
    Closed CircuitState = iota
    Open
    HalfOpen
)

func (cb *CircuitBreaker) CanExecute() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    
    switch cb.state {
    case Closed:
        return true
    case Open:
        if time.Since(cb.lastFailureTime) > cb.resetTimeout {
            cb.state = HalfOpen
            return true
        }
        return false
    case HalfOpen:
        return true
    default:
        return false
    }
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.failureCount = 0
    cb.state = Closed
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.failureCount++
    cb.lastFailureTime = time.Now()
    
    if cb.failureCount >= int64(cb.maxFailures) {
        cb.state = Open
    }
}
```

### 4. 실패 재시도 메커니즘

```go
// src/events/retryHandler.go
type RetryHandler struct {
    dlq           *DeadLetterQueue
    eventBus      *EventBus
    retrySchedule []time.Duration // [1s, 5s, 30s, 5m]
}

func (rh *RetryHandler) ProcessFailures() {
    for failure := range rh.dlq.failedEvents {
        if failure.RetryCount < len(rh.retrySchedule) {
            // 지연 후 재시도
            time.AfterFunc(rh.retrySchedule[failure.RetryCount], func() {
                rh.retryFailedEvent(failure)
            })
        } else {
            // 최대 재시도 횟수 초과, 영구 저장
            rh.dlq.storage.PersistPermanentFailure(failure)
            
            // 관리자에게 알림
            rh.eventBus.SystemAlertEvent.Publish(events.Event[dto.SystemAlertData]{
                Data: dto.SystemAlertData{
                    Level:   "CRITICAL",
                    Message: fmt.Sprintf("Permanent event failure: %s", failure.FailedEvent),
                    Details: failure,
                },
            })
        }
    }
}
```

### 5. 모니터링 및 메트릭

```go
// src/events/eventMetrics.go
type EventMetrics struct {
    totalEvents    int64
    failedEvents   int64
    retryEvents    int64
    eventLatency   map[string]time.Duration
    mutex          sync.RWMutex
}

func (em *EventMetrics) RecordEventProcessed(eventType string, duration time.Duration, success bool) {
    em.mutex.Lock()
    defer em.mutex.Unlock()
    
    atomic.AddInt64(&em.totalEvents, 1)
    
    if !success {
        atomic.AddInt64(&em.failedEvents, 1)
    }
    
    em.eventLatency[eventType] = duration
}

func (em *EventMetrics) GetFailureRate() float64 {
    total := atomic.LoadInt64(&em.totalEvents)
    failed := atomic.LoadInt64(&em.failedEvents)
    
    if total == 0 {
        return 0
    }
    
    return float64(failed) / float64(total)
}
```

---

## 구현 단계

### Phase 1: 기본 실패 처리 강화
1. Dead Letter Queue 구현
2. 실패 이벤트 타입 추가
3. 기존 TypedBus에 실패 처리 로직 추가

### Phase 2: 고급 복구 메커니즘
1. Circuit Breaker 패턴 구현
2. 재시도 로직 추가
3. 실패 영구 저장 메커니즘

### Phase 3: 모니터링 및 알림
1. 메트릭 수집 시스템
2. 대시보드 구현
3. 알림 시스템 통합

---

## 설정 추가

```toml
# env.toml에 추가
[event_system]
# Dead Letter Queue 설정
dlq_buffer_size = 1000
max_retries = 3
retry_delays = ["1s", "5s", "30s", "5m"]

# Circuit Breaker 설정
circuit_breaker_enabled = true
max_failures = 5
reset_timeout = "30s"

# 모니터링 설정
metrics_enabled = true
metrics_interval = "10s"
alert_threshold = 0.1  # 10% 실패율에서 알림
```

---

## 예상 효과

### 1. 시스템 안정성 향상
- 단일 실패가 전체 시스템에 미치는 영향 최소화
- 자동 복구 메커니즘으로 일시적 장애 극복

### 2. 운영 가시성 확보
- 실시간 실패 모니터링
- 장애 패턴 분석을 통한 사전 예방

### 3. 개발 생산성 향상
- 명확한 실패 원인 추적
- 재현 가능한 테스트 환경

### 4. 사용자 경험 개선
- 부분 실패 시에도 기능 계속 사용 가능
- 빠른 장애 복구

---

## 주의사항

1. **성능 오버헤드**: 추가 처리 로직으로 인한 지연 시간 증가
2. **메모리 사용량**: DLQ 및 메트릭 저장으로 인한 메모리 사용량 증가
3. **복잡성 증가**: 디버깅 및 유지보수 복잡도 상승
4. **설정 관리**: 다양한 임계값과 타이밍 설정의 최적화 필요

---

*작성일: 2025-09-05*  
*작성자: Claude Code Assistant*