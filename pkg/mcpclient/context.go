package mcpclient

import "context"

type managerKey struct{}

func ManagerToContext(ctx context.Context, manager Manager) context.Context {
	return context.WithValue(ctx, managerKey{}, manager)
}

func ManagerFromContext(ctx context.Context) (Manager, bool) {
	manager, ok := ctx.Value(managerKey{}).(Manager)
	return manager, ok
}
