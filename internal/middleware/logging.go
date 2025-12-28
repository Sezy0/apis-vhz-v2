package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// Logging is a middleware that logs HTTP requests.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		
		// Status Color
		statusColor := ColorGreen
		if wrapped.statusCode >= 400 && wrapped.statusCode < 500 {
			statusColor = ColorYellow
		} else if wrapped.statusCode >= 500 {
			statusColor = ColorRed
		}

		// Method Color
		methodColor := ColorBlue
		switch r.Method {
		case "GET":
			methodColor = ColorCyan
		case "POST":
			methodColor = ColorGreen
		case "PUT":
			methodColor = ColorYellow
		case "DELETE":
			methodColor = ColorRed
		}

		// Simplify RemoteAddr (remove port)
		clientIP := r.RemoteAddr
		if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
			clientIP = clientIP[:idx]
		}
		
		// Format duration
		durStr := duration.String()
		if duration < time.Millisecond {
			durStr = fmt.Sprintf("%dµs", duration.Microseconds())
		} else if duration < time.Second {
			durStr = fmt.Sprintf("%.2fms", float64(duration.Microseconds())/1000)
		}

		// Concise Log Format: [METHOD] /path CODE DURATION IP
		// Remove date/time prefix from log.Printf by formatting message directly
		// Note: standard logger adds timestamp automatically. We'll use a custom format.
		// Since we can't easily change global logger flags here without affecting everything, 
		// we craft the message to be clean.
		
		log.Printf(
			"%s%3s%s %s %s%3d%s %s %s",
			methodColor, r.Method, ColorReset,
			r.URL.Path,
			statusColor, wrapped.statusCode, ColorReset,
			durStr,
			clientIP,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
