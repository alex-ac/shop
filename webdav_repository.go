package shop

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

func init() {
	RepositoryFactories["webdav"] = NewWebDAVRepository
	RepositoryFactories["https+webdav"] = NewWebDAVRepository
}

type WebDAVRepository struct {
	config  RepositoryConfig
	baseURL *url.URL
	client  *http.Client
}

func NewWebDAVRepository(ctx context.Context, cfg RepositoryConfig) (Repository, error) {
	url, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}

	url.Scheme = map[string]string{
		"webdav":       "http",
		"https+webdav": "https",
	}[url.Scheme]

	return &WebDAVRepository{
		config:  cfg,
		baseURL: url,
		client:  http.DefaultClient,
	}, nil
}

func (r *WebDAVRepository) GetConfig() RepositoryConfig {
	return r.config
}

func (r *WebDAVRepository) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	url := r.baseURL.JoinPath(key)
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	rsp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		if rsp.Body != nil {
			rsp.Body.Close()
		}
		return nil, err
	}

	return rsp.Body, err
}

func (*WebDAVRepository) Put(ctx context.Context, key string, body io.ReadCloser) error {
	return nil
}

func (r *WebDAVRepository) GetJSON(ctx context.Context, key string, output any) (err error) {
	body, err := r.Get(ctx, key)
	if body != nil {
		defer body.Close()
	}
	if err == nil {
		decoder := json.NewDecoder(body)
		err = decoder.Decode(output)
	}
	return
}

func (*WebDAVRepository) PutJSON(ctx context.Context, key string, input any) error {
	return nil
}

func (*WebDAVRepository) List(ctx context.Context, prefix string) Paginator {
	return nil
}

func (*WebDAVRepository) Copy(ctx context.Context, srcKey, dstKey string) error {
	return nil
}

func (*WebDAVRepository) Delete(ctx context.Context, key string) error {
	return nil
}

var _ Repository = (*WebDAVRepository)(nil)
