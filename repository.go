package shop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/hashicorp/go-multierror"
)

const (
	RepositoryManifestKey = "shop-repository.json"
)

var (
	RepositoryFactories      = map[string]func(context.Context, RepositoryConfig) (RepositoryFS, error){}
	ErrRepoWriteIsNotAllowed = errors.New("Write to the repository is not enabled in configuration")
	ErrRepoAdminIsNotAllowed = errors.New("Admin action on the repository is not enabled in configuration")
)

type Entry struct {
	Key      string
	IsPrefix bool
}

type RepositoryManifest struct {
	ApiVersion  string        `json:"api_version"`
	URL         string        `json:"url"`
	Name        string        `json:"name"`
	ReadOnlyURL string        `json:"readonly_url,omitempty"`
	UpdatedAt   UnixTimestamp `json:"updated_at"`
}

type Repository interface {
	GetConfig() RepositoryConfig

	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Put(ctx context.Context, key string, body io.Reader) error

	GetJSON(ctx context.Context, key string, output any) error
	PutJSON(ctx context.Context, key string, input any) error

	List(ctx context.Context, prefix string) Cursor[Entry]

	EnsurePrefix(ctx context.Context, key string) error
	Delete(ctx context.Context, key string) error

	GetManifest(ctx context.Context) (RepositoryManifest, error)
	PutManifest(ctx context.Context, manifest RepositoryManifest) error

	ResourceExists(ctx context.Context, key string) (bool, error)
}

type RepositoryFS interface {
	Read(context.Context, string) ([]byte, error)
	Write(context.Context, string, []byte) error
	Open(context.Context, string) (io.ReadCloser, error)
	Create(context.Context, string) (io.WriteCloser, error)
	MakeDir(context.Context, string) error
	ListDir(context.Context, string) Cursor[Entry]
	Remove(context.Context, string) error
	Exists(context.Context, string) (bool, error)
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
	return r.fs.Open(ctx, key)
}

func (r repositoryImpl) Put(ctx context.Context, key string, body io.Reader) (err error) {
	w, err := r.fs.Create(ctx, key)
	if err != nil {
		return
	}

	defer func() {
		err = multierror.Append(err, w.Close()).ErrorOrNil()
	}()

	_, err = io.Copy(w, body)
	return
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
	return r.fs.ListDir(ctx, prefix)
}

func (r repositoryImpl) EnsurePrefix(ctx context.Context, prefix string) error {
	if !r.cfg.Write {
		return fmt.Errorf("%w: %s / %s", ErrRepoAdminIsNotAllowed, r.cfg.URL, prefix)
	}
	return r.fs.MakeDir(ctx, prefix)
}

func (r repositoryImpl) Delete(ctx context.Context, key string) error {
	if !r.cfg.Write {
		return fmt.Errorf("%w: %s / %s", ErrRepoAdminIsNotAllowed, r.cfg.URL, key)
	}
	return r.fs.Remove(ctx, key)
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

func (r repositoryImpl) ResourceExists(ctx context.Context, key string) (bool, error) {
	return r.fs.Exists(ctx, key)
}
