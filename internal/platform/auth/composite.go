package auth

import "context"

type CompositeVerifier struct {
	verifiers []Verifier
}

func NewCompositeVerifier(verifiers ...Verifier) *CompositeVerifier {
	return &CompositeVerifier{verifiers: verifiers}
}

func (verifier *CompositeVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	for _, candidate := range verifier.verifiers {
		principal, err := candidate.Verify(ctx, token)
		if err == nil {
			return principal, nil
		}
	}
	return Principal{}, ErrUnauthorized
}
