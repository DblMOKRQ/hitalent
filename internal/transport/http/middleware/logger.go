package middleware

import (
	"context"
	"hitalent/internal/transport/http/constants"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter - это обертка над стандартным http.ResponseWriter,
// чтобы мы могли запомнить код ответа (status code)
type responseWriter struct {
	http.ResponseWriter
	status int
}

// Переопределяем метод WriteHeader, чтобы сохранить статус
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestLog := log.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
			)

			ctx := context.WithValue(r.Context(), constants.LoggerKey, requestLog)
			r = r.WithContext(ctx)

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			latency := time.Since(start)

			switch {
			case rw.status >= http.StatusBadRequest && rw.status < http.StatusInternalServerError:
				requestLog.Warn("Request completed with client error", zap.Int("status", rw.status), zap.Duration("latency", latency))
			case rw.status >= http.StatusInternalServerError:
				requestLog.Error("Request completed with server error", zap.Int("status", rw.status), zap.Duration("latency", latency))
			default:
				requestLog.Info("Request completed successfully", zap.Int("status", rw.status), zap.Duration("latency", latency))
			}
		})
	}
}

// GetLogger - хэлпер для того, чтобы доставать логгер внутри хэндлеров
func GetLogger(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(constants.LoggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}
