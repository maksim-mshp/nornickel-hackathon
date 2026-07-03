package auth

import "context"

type Verifier interface {
	Verify(ctx context.Context, token string) (Principal, error)
}
