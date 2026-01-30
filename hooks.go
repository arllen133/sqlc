package sqlc

import (
	"context"
)

// Lifecycle Interfaces
type BeforeCreateInterface interface {
	BeforeCreate(context.Context) error
}

type AfterCreateInterface interface {
	AfterCreate(context.Context) error
}

type BeforeUpdateInterface interface {
	BeforeUpdate(context.Context) error
}

type AfterUpdateInterface interface {
	AfterUpdate(context.Context) error
}

type BeforeDeleteInterface interface {
	BeforeDelete(context.Context) error
}

type AfterDeleteInterface interface {
	AfterDelete(context.Context) error
}

func triggerBeforeCreate(ctx context.Context, model any) error {
	if m, ok := model.(BeforeCreateInterface); ok {
		return m.BeforeCreate(ctx)
	}
	return nil
}

func triggerAfterCreate(ctx context.Context, model any) error {
	if m, ok := model.(AfterCreateInterface); ok {
		return m.AfterCreate(ctx)
	}
	return nil
}

func triggerBeforeUpdate(ctx context.Context, model any) error {
	if m, ok := model.(BeforeUpdateInterface); ok {
		return m.BeforeUpdate(ctx)
	}
	return nil
}

func triggerAfterUpdate(ctx context.Context, model any) error {
	if m, ok := model.(AfterUpdateInterface); ok {
		return m.AfterUpdate(ctx)
	}
	return nil
}

func triggerBeforeDelete(ctx context.Context, model any) error {
	if m, ok := model.(BeforeDeleteInterface); ok {
		return m.BeforeDelete(ctx)
	}
	return nil
}

func triggerAfterDelete(ctx context.Context, model any) error {
	if m, ok := model.(AfterDeleteInterface); ok {
		return m.AfterDelete(ctx)
	}
	return nil
}
