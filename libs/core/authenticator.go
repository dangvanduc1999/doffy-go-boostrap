package core

import "context"

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (bool, error)
	Assert(ctx context.Context, token string) (bool, error)
}



