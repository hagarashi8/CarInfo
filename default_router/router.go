package defaultrouter

import (
	"effMob/api"
	"net/http"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func SetupMux(client *api.APIClient, db *gorm.DB, log *logrus.Logger)(http.Handler){
	lmw := LogMiddleware(log)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /car", GetCarHandler(client, db, log))
	mux.Handle("POST /car", AddCarHandler(client, db, log))
	mux.Handle("DELETE /car", DeleteCarHandler(client, db, log))
	mux.Handle("PUT /car", PutCarHandler(client, db, log))
	wmux := lmw(mux)
	return wmux
}

func LogMiddleware(log *logrus.Logger) (func(http.Handler) http.Handler) {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lrw := NewLoggingResponseWriter(w)
			h.ServeHTTP(w, r)
			code := lrw.statusCode
			level := logrus.InfoLevel
			switch code / 100 {
			case 4:
			level = logrus.WarnLevel
			case 5:
			level = logrus.ErrorLevel
			}
			log.Logf(level, "%s %s %d\n", r.Method, r.URL.Path, code)
		})
	}
}
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
