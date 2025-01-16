package finalizer

import (
	"context"
	"github.com/hyle-team/tss-svc/internal/tss/session"
)

type FinalizerSession struct {
	finalizer interface {
		Run(ctx context.Context) error
	}
}

func (f *FinalizerSession) Run(ctx context.Context) error {
	boundedCtx, cancel := context.WithTimeout(ctx, session.BoundaryFinalizeSession)
	defer cancel()
	return f.finalizer.Run(boundedCtx)
}
