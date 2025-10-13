package http

import (
	"net/http"

	"github.com/Bridgeless-Project/tss-svc/internal/api/ctx"
	"gitlab.com/distributed_lab/ape"
)

func Health(w http.ResponseWriter, r *http.Request) {
	response := ctx.HealthChecker(r.Context()).Check()

	if !response.Ok {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	ape.Render(w, response)
}
