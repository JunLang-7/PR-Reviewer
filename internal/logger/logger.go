package logger

import (
	"log"
	"net/http"
	"time"
)

// RequestLogger wraps an http.Handler and logs every incoming request.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[http] %s %s %d (%s)", r.Method, r.URL.Path, 0, time.Since(start))
	})
}

// Log logs an info message.
func Info(format string, v ...interface{}) {
	log.Printf("[info] "+format, v...)
}

// Log logs a debug message.
func Debug(format string, v ...interface{}) {
	log.Printf("[debug] "+format, v...)
}
