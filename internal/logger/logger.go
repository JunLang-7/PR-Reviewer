package logger

import (
	"log"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// RequestLogger wraps an http.Handler and logs every incoming request.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		log.Printf("[http] %s %s → %d (%s)", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

func Info(format string, v ...interface{}) {
	log.Printf("[info] "+format, v...)
}

func Debug(format string, v ...interface{}) {
	log.Printf("[debug] "+format, v...)
}
