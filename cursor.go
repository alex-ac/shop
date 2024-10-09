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

type SliceCursor[T any] struct {
	items []T
}

func NewSliceCursor[T any](items []T) Cursor[T] {
	return &SliceCursor[T]{items}
}

func (c *SliceCursor[T]) GetNext(context.Context) (item *T, err error) {
	if len(c.items) > 0 {
		item = &c.items[0]
		c.items = c.items[1:]
	}
	return
}
