package shop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"
)

const (
	RepositoryManifestKey = "shop-repository.json"
)

var (
	RepositoryFactories      = map[string]func(context.Context, RepositoryConfig) (RepositoryFS, error){}
	ErrUnimplemented         = errors.New("Unimplemented")
	ErrRepoWriteIsNotAllowed = errors.New("Write to the repository is not enabled in configuration")
	ErrRepoAdminIsNotAllowed = errors.New("Admin action on the repository is not enabled in configuration")
)

type Entry struct {
	Key      string
	IsPrefix bool
}

type Repository interface {
	GetConfig() RepositoryConfig

	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Put(ctx context.Context, key string, body io.ReadCloser) error

	GetJSON(ctx context.Context, key string, output any) error
	PutJSON(ctx context.Context, key string, input any) error

	List(ctx context.Context, prefix string) Cursor[Entry]

	Copy(ctx context.Context, srcKey, dstKey string) error
	Delete(ctx context.Context, key string) error

	GetManifest(ctx context.Context) (RepositoryManifest, error)
	PutManifest(ctx context.Context, manifest RepositoryManifest) error
}

type RepositoryFS interface {
	Read(context.Context, string) ([]byte, error)
	Write(context.Context, string, []byte) error
}

type repositoryImpl struct {
	cfg RepositoryConfig
	fs  RepositoryFS
}

func NewRepository(ctx context.Context, cfg RepositoryConfig) (repository Repository, err error) {
	url, err := url.Parse(cfg.URL)
	if err != nil {
		return
	}

	var fs RepositoryFS
	if factory, ok := RepositoryFactories[url.Scheme]; ok {
		fs, err = factory(ctx, cfg)
	} else {
		err = fmt.Errorf("Unknown url schema: %s", url.Scheme)
	}

	return repositoryImpl{
		cfg: cfg,
		fs:  fs,
	}, nil
}

func (r repositoryImpl) GetConfig() RepositoryConfig {
	return r.cfg
}

func (r repositoryImpl) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, ErrUnimplemented
}

func (r repositoryImpl) Put(ctx context.Context, key string, body io.ReadCloser) error {
	return ErrUnimplemented
}

func (r repositoryImpl) GetJSON(ctx context.Context, key string, output any) error {
	data, err := r.fs.Read(ctx, key)
	if err == nil {
		err = json.Unmarshal(data, output)
	}
	return err
}

func (r repositoryImpl) PutJSON(ctx context.Context, key string, input any) error {
	if !r.cfg.Write {
		return fmt.Errorf("%w: %s / %s", ErrRepoWriteIsNotAllowed, r.cfg.URL, key)
	}
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return r.fs.Write(ctx, key, data)
}

func (r repositoryImpl) List(ctx context.Context, prefix string) Cursor[Entry] {
	return NewErrorCursor[Entry](ErrUnimplemented)
}

func (r repositoryImpl) Copy(ctx context.Context, srcKey, dstKey string) error {
	return ErrUnimplemented
}

func (r repositoryImpl) Delete(ctx context.Context, key string) error {
	return ErrUnimplemented
}

func (r repositoryImpl) GetManifest(ctx context.Context) (manifest RepositoryManifest, err error) {
	err = r.GetJSON(ctx, RepositoryManifestKey, &manifest)
	return
}

func (r repositoryImpl) PutManifest(ctx context.Context, manifest RepositoryManifest) error {
	manifest.UpdatedAt = UnixTimestamp{time.Now()}
	if !r.cfg.Admin {
		return fmt.Errorf("Update manifest: %w: %s", ErrRepoAdminIsNotAllowed, r.cfg.URL)
	}
	return r.PutJSON(ctx, RepositoryManifestKey, manifest)
}
