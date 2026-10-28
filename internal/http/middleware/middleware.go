package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	api "github.com/kokos46/tripgo/internal/generated"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseWriter) Write(value []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(value)
}
func Observability(tracer trace.Tracer, meter metric.Meter, logger *slog.Logger) func(http.Handler) http.Handler {
	duration, _ := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	active, _ := meter.Int64UpDownCounter("http.server.active_requests")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			route := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
				route = rc.RoutePattern()
			}
			ctx, span := tracer.Start(r.Context(), r.Method+" "+route)
			r = r.WithContext(ctx)
			active.Add(ctx, 1, metric.WithAttributes(attribute.String("http.request.method", r.Method), attribute.String("http.route", route)))
			writer := &responseWriter{ResponseWriter: w}
			defer func() {
				active.Add(ctx, -1, metric.WithAttributes(attribute.String("http.request.method", r.Method), attribute.String("http.route", route)))
				elapsed := time.Since(started).Seconds()
				status := writer.status
				if status == 0 {
					status = http.StatusOK
				}
				duration.Record(ctx, elapsed, metric.WithAttributes(attribute.String("http.request.method", r.Method), attribute.String("http.route", route), attribute.Int("http.response.status_code", status)))
				attrs := []any{"service", "trip-service", "method", r.Method, "route", route, "status", status, "duration_ms", time.Since(started).Milliseconds()}
				spanContext := trace.SpanContextFromContext(ctx)
				if spanContext.IsValid() {
					attrs = append(attrs, "trace_id", spanContext.TraceID().String(), "span_id", spanContext.SpanID().String())
				}
				logger.LogAttrs(ctx, logLevel(status), "request handled", attrsToAttrs(attrs)...)
				span.End()
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic recovered", "panic", recovered, "stack", string(debug.Stack()))
					instance := r.URL.Path
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(api.Problem{
						Type:     "https://tripgo.example/problems/internal-error",
						Title:    "Internal Server Error",
						Status:   http.StatusInternalServerError,
						Code:     "internal_error",
						Instance: &instance,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
func logLevel(status int) slog.Level {
	if status >= 500 {
		return slog.LevelError
	}
	return slog.LevelInfo
}
func attrsToAttrs(values []any) []slog.Attr {
	result := make([]slog.Attr, 0, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		result = append(result, slog.Any(values[index].(string), values[index+1]))
	}
	return result
}
