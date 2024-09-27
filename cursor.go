package shop

import (
	"context"
)

type Cursor[T any] interface {
	GetNext(context.Context) (*T, error)
}

type ErrorCursor[T any] struct {
	error
}

func NewErrorCursor[T any](err error) Cursor[T] {
	return ErrorCursor[T]{err}
}

func (c ErrorCursor[T]) GetNext(context.Context) (*T, error) {
	return nil, c
}
