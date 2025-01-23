package middlewares

import (
	"context"
	"net/http"
)

// HijackedConnectionCloser is used to notify hijacked connections about program termination.
// By default, `server.Shutdown()` does not not attempt to close nor wait for hijacked connections such as WebSockets.
func HijackedConnectionCloser(appCtx context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			closerCtx, cancel := context.WithCancel(r.Context())
			defer cancel()

			go func() {
				select {
				case <-appCtx.Done():
					cancel()
				case <-closerCtx.Done():
					// returning from goroutine
				}
			}()

			next.ServeHTTP(w, r.WithContext(closerCtx))
		})
	}
}
