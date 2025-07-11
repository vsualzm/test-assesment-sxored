# UnderwritingService with Caching and Circuit Breaker
## 🧩 Overview

This project is a sample implementation of **efficient service-to-service communication** in a microservices context. Specifically, it simulates how `UnderwritingService` interacts with `CreditService` while handling:

- High request volume (500+ loan evaluations per minute)
- Reducing dependency on external service by caching credit scores
- Ensuring resilience during `CreditService` downtime using circuit breaker and fallback logic

---

## 📚 Problem Context
Given three microservices:
- `ApplicationService`
- `CreditService`
- `UnderwritingService`

The `UnderwritingService` needs to:
- Validate credit scores (via `CreditService`)
- Perform risk assessment based on credit data

---

## ✅ Solution Summary

### 1. Efficient Communication Pattern

To handle high volume:
- UnderwritingService performs **synchronous call** to `CreditService`
- Enhanced with:
  - **In-memory caching** (to avoid redundant calls)
  - **Circuit breaker** (to stop overwhelming `CreditService` when it's down)

---

### 2. Caching Mechanism (24h validity)

```go
type creditCacheItem struct {
    report   *CreditReport
    cachedAt time.Time
}

type UnderwritingServiceImpl struct {
    creditSvc CreditService
    cache     sync.Map // key: ssn, value: creditCacheItem
    cacheTTL  time.Duration
}
```

Implementation:

```go
func (u *UnderwritingServiceImpl) getCachedCreditScore(ctx context.Context, ssn string) (*CreditReport, error) {
    if val, ok := u.cache.Load(ssn); ok {
        item := val.(creditCacheItem)
        if time.Since(item.cachedAt) < u.cacheTTL {
            return item.report, nil // ✅ use cache
        }
    }

    report, err := u.creditSvc.GetCreditScore(ctx, ssn)
    if err != nil {
        if val, ok := u.cache.Load(ssn); ok {
            item := val.(creditCacheItem)
            return item.report, nil // ⚠️ fallback to stale cache
        }
        return nil, fmt.Errorf("credit service unavailable and no cached data")
    }

    u.cache.Store(ssn, creditCacheItem{
        report:   report,
        cachedAt: time.Now(),
    })
    return report, nil
}
```

---

### 3. Handle Downtime with Circuit Breaker + Stale Cache

```go
import "github.com/sony/gobreaker"

type UnderwritingServiceImpl struct {
    creditSvc CreditService
    breaker   *gobreaker.CircuitBreaker
    ...
}

func NewUnderwritingService(creditSvc CreditService) *UnderwritingServiceImpl {
    cbSettings := gobreaker.Settings{
        Name:        "CreditServiceCB",
        MaxRequests: 3,
        Interval:    30 * time.Second,
        Timeout:     15 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures >= 5
        },
    }

    return &UnderwritingServiceImpl{
        creditSvc: creditSvc,
        breaker:   gobreaker.NewCircuitBreaker(cbSettings),
        ...
    }
}
```

Usage:

```go
func (u *UnderwritingServiceImpl) getCreditScoreWithBreaker(ctx context.Context, ssn string) (*CreditReport, error) {
    result, err := u.breaker.Execute(func() (interface{}, error) {
        return u.creditSvc.GetCreditScore(ctx, ssn)
    })
    if err != nil {
        if val, ok := u.cache.Load(ssn); ok {
            item := val.(creditCacheItem)
            return item.report, nil // fallback to stale cache
        }
        return nil, fmt.Errorf("credit service down and no cache")
    }

    report := result.(*CreditReport)
    u.cache.Store(ssn, creditCacheItem{
        report:   report,
        cachedAt: time.Now(),
    })
    return report, nil
}
```

---

### 4. Application Evaluation Logic
```go
func (u *UnderwritingServiceImpl) EvaluateApplication(ctx context.Context, app *LoanApplication) (*UnderwritingDecision, error) {
    score, err := u.getCreditScoreWithBreaker(ctx, app.SSN)
    if err != nil {
        return nil, err
    }

    decision := &UnderwritingDecision{}
    if score.Score > 700 {
        decision.Status = "Approved"
    } else {
        decision.Status = "Manual Review"
    }
    return decision, nil
}
```

---

## 🧠 Architecture Summary

```
[UnderwritingService]
   ↳ Check cache for SSN
   ↳ If cache hit: return score
   ↳ Else: call CreditService via Circuit Breaker
       ↳ If success: store to cache and return
       ↳ If failed:
           ↳ Use stale cache if available
           ↳ Return error if no data
   ↳ Evaluate application based on credit score
```

---

## 🧪 Testing

To simulate service up/down and observe cache & fallback:

```go
func main() {
    ctx := context.Background()
    creditService := &CreditServiceMock{fail: false}
    underwriting := NewUnderwritingService(creditService)

    app := &LoanApplication{SSN: "123-45-6789"}
    decision, _ := underwriting.EvaluateApplication(ctx, app)
    fmt.Println("✅ Decision:", decision)

    // simulate service failure
    creditService.fail = true
    decision, err := underwriting.EvaluateApplication(ctx, app)
    fmt.Println("⚠️  Fallback Decision:", decision, "Error:", err)
}
```

---

## 📦 Dependencies

- `github.com/sony/gobreaker`: Circuit breaker implementation
- `sync.Map`: Thread-safe in-memory cache (can be replaced with LRU/Redis/etc.)

---

## 🚀 Future Improvements

- Use distributed cache (e.g., Redis) in production
- Add logging and metrics (cache hits, fallback count)
- Use background refresh for expiring cache (optional)
