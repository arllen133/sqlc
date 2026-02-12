---
trigger: always_on
---

# Modern Golang Agent Rules & Best Practices

## 1. Core Principles

* **Simplicity First**: Code must be intuitive and readable. Avoid over-engineering and unnecessary abstractions.
* **Modern Features**: Fully embrace **Go 1.18+ Generics** and **Go 1.21+ Standard Library** enhancements (`slog`, `slices`, `maps`).
* **Type Safety**: Minimize the use of `interface{}` or `any`. Leverage strong typing to ensure compile-time safety.
* **Zero Tolerance for Deprecated APIs**: Never use functions marked as `Deprecated`. (e.g., use `os.ReadFile` instead of `ioutil.ReadFile`).

---

## 2. Architecture & Design Patterns

* **Clean Architecture**: Strictly separate layers to decouple business logic from technical implementation (frameworks, DBs):
* `cmd/`: Application entry points.
* `internal/`: Core business logic (private; cannot be imported by external projects).
* `pkg/`: Exportable utility packages.
* `api/`: Transport layer definitions (gRPC/REST handlers).


* **Interface-Driven Development**:
* **Dependency Injection**: Inject dependencies via constructors; avoid global variables.
* **Programming to Interfaces**: Public functions should accept interfaces rather than concrete implementations.
* **Composition Over Inheritance**: Use struct embedding for functional reuse.



---

## 3. Coding Standards & Idioms

* **Generics (Go 1.18+)**:
* Use only for generic containers or utility functions where logic is type-agnostic.
* If logic requires specific methods, prioritize Interface constraints over empty generics.


* **Standard Library Preference**:
* **Logging**: Use `log/slog` for structured logging. Always propagate `context`.
* **Data Ops**: Use the `slices` (Sort, Contains) and `maps` (Keys, Values) packages for common operations.


* **Error Handling**:
* **Wrapping**: Use `fmt.Errorf("...: %w", err)` to preserve context.
* **Checking**: Never use `==` for error comparison; use `errors.Is()` or `errors.As()`.
* **Explicitness**: Always handle returned errors; never ignore them silently.



---

## 4. Concurrency & Context

* **Context Propagation**: `ctx context.Context` must be the first parameter for all blocking, I/O, or long-running functions.
* **Structured Concurrency**:
* Avoid orphaned goroutines. Use `golang.org/x/sync/errgroup` to manage lifecycles.
* Use `context` cancellation to prevent goroutine leaks and deadlocks.


* **Concurrency Safety**: Guard shared states with channels or `sync` primitives. Strictly avoid Data Races.

---

## 5. Performance Optimization

* **Memory Allocation**: Initialize slices with capacity where possible: `make([]T, 0, cap)`.
* **String Manipulation**: Always use `strings.Builder` for frequent concatenations in loops.
* **Resource Reuse**: Use `sync.Pool` for high-frequency, short-lived objects.
* **Benchmarking**: Profile critical paths with `go test -bench` to avoid premature optimization.

---

## 6. Observability (OpenTelemetry)

* **Distributed Tracing**:
* Propagate Trace Context across service boundaries (HTTP, gRPC, DB).
* Use `otel.Tracer` to create Spans. Record key attributes (User ID, Params) and errors.


* **Metrics**:
* Use `otel.Meter` to collect **RED** metrics (Requests, Errors, Duration).
* Monitor Service Level Indicators (SLIs) like p99 latency and throughput.


* **Log Correlation**: Inject `trace_id` and `span_id` into structured JSON logs for unified debugging.

---

## 7. Security & Resilience

* **Defensive Programming**: Rigorously validate and sanitize all external inputs.
* **Stability Patterns**: Implement **Timeouts**, **Retries (with Exponential Backoff)**, and **Circuit Breakers** for all external calls.
* **Rate Limiting**: Implement service-level rate limiting (use Redis for distributed scenarios).

---

## 8. Testing & Tooling

* **Test-Driven Development (TDD)**:
* Use **Table-Driven Tests** for comprehensive edge-case coverage.
* Ensure high coverage for exported functions with `go test -cover`.


* **Mocking**: Use lightweight mocks via interfaces (hand-written or generated).
* **Toolchain**:
* Mandatory use of `golangci-lint` (enable `revive`, `staticcheck`, `govet`).
* Automate formatting with `go fmt` and `goimports`.


* **Documentation**: Annotate all public APIs with GoDoc comments.

---

## 9. Deprecated API Replacement Guide

| Deprecated API | Modern Replacement |
| --- | --- |
| `ioutil.ReadAll` | `io.ReadAll` |
| `ioutil.ReadFile` | `os.ReadFile` |
| `ioutil.WriteFile` | `os.WriteFile` |
| `ioutil.TempFile` | `os.CreateTemp` |
| `math/rand` (for security) | `crypto/rand` |
| `errors` (custom simple) | `fmt.Errorf("...: %w", err)` |

---
