package shop

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

var (
	RepositoryFactories = map[string]func(context.Context, RepositoryConfig) (Repository, error){}
)

type Paginator interface {
	HasMore() bool
	GetNextKey() (string, error)
}

type Repository interface {
	GetConfig() RepositoryConfig

	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Put(ctx context.Context, key string, body io.ReadCloser) error

	GetJSON(ctx context.Context, key string, output any) error
	PutJSON(ctx context.Context, key string, input any) error

	List(ctx context.Context, prefix string) Paginator

	Copy(ctx context.Context, srcKey, dstKey string) error
	Delete(ctx context.Context, key string) error
}

func NewRepository(ctx context.Context, cfg RepositoryConfig) (repository Repository, err error) {
	url, err := url.Parse(cfg.URL)
	if err != nil {
		return
	}

	if factory, ok := RepositoryFactories[url.Scheme]; ok {
		repository, err = factory(ctx, cfg)
	} else {
		err = fmt.Errorf("Unknown url schema: %s", url.Scheme)
	}
	return
}
