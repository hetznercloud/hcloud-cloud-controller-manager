package cache

import (
	"context"
)

type key struct{}

var subsystemKey = key{}

func SetSubsystem(ctx context.Context, subsystem string) context.Context {
	return context.WithValue(ctx, subsystemKey, subsystem)
}

func GetSubsystem(ctx context.Context) string {
	result, ok := ctx.Value(subsystemKey).(string)
	if !ok {
		return "none"
	}
	return result
}
