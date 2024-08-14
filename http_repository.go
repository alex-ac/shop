package shop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func init() {
	RepositoryFactories["http"] = NewHTTPRepository
	RepositoryFactories["https"] = NewHTTPRepository
}

type HTTPRepository struct {
	config  RepositoryConfig
	baseURL *url.URL
	client  *http.Client
}

func NewHTTPRepository(ctx context.Context, cfg RepositoryConfig) (Repository, error) {
	url, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}

	r := &HTTPRepository{
		config:  cfg,
		baseURL: url,
		client:  http.DefaultClient,
	}

	var manifest RegistryManifest
	err = r.GetJSON(ctx, RegistryManifestKey, &manifest)
	if err != nil {
		return nil, err
	}

	url, err = url.Parse(manifest.RootRepo.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: while trying to parse %s", err, manifest.RootRepo.URL)
	}

	if url.Scheme == "http" || url.Scheme == "https" {
		return nil, fmt.Errorf("Repo manifest MUST NOT contain http url: %s", manifest.RootRepo.URL)
	}

	cfg.URL = manifest.RootRepo.URL
	return NewRepository(ctx, cfg)
}

func (r *HTTPRepository) Get(ctx context.Context, key string) (io.ReadCloser, error) {
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

func (r *HTTPRepository) GetJSON(ctx context.Context, key string, output any) (err error) {
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
