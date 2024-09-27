package shop

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
)

func init() {
	RepositoryFactories["file"] = NewFileFS
}

type FileFS struct {
	cfg  RepositoryConfig
	path string
}

func NewFileFS(ctx context.Context, cfg RepositoryConfig) (RepositoryFS, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}

	return FileFS{
		cfg:  cfg,
		path: filepath.Join(u.Host, u.Path),
	}, nil
}

func (f FileFS) Read(ctx context.Context, path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(f.path, path))
}

func (f FileFS) Write(ctx context.Context, path string, data []byte) error {
	return os.WriteFile(filepath.Join(f.path, path), data, 0666)
}
