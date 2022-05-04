package util

import "context"

type CtxKey struct{}


// GetCallerFromContext gets the caller value from the context.
func GetProtocolFromContext(ctx context.Context) (string, bool) {
    caller, ok := ctx.Value(CtxKey{}).(string)
    return caller, ok
}
