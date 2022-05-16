package quicshim

import "context"

func cancelWhenClose[T any](ctx context.Context, ch <-chan T) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		select {
		case <-ch:
		case <-ctx.Done():
		}
		cancel()
	}()

	return ctx, cancel
}
