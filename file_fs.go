package shop

import (
	"context"
	"errors"
	"io"
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

func (f FileFS) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(f.path, path))
}

func (f FileFS) Create(ctx context.Context, path string) (io.WriteCloser, error) {
	return os.OpenFile(filepath.Join(f.path, path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
}

func (f FileFS) MakeDir(ctx context.Context, path string) error {
	return os.MkdirAll(filepath.Join(f.path, path), 0777)
}

type fileFSCursor struct {
	cancel  func()
	file    *os.File
	entries []os.FileInfo
}

func (f FileFS) ListDir(ctx context.Context, path string) Cursor[Entry] {
	file, err := os.Open(filepath.Join(f.path, path))
	if err != nil {
		return NewErrorCursor[Entry](err)
	}

	ctx, cancel := context.WithCancel(ctx)
	c := &fileFSCursor{
		cancel: cancel,
		file:   file,
	}

	go c.cleanup(file, ctx)
	return c
}

func (c *fileFSCursor) GetNext(context.Context) (ret *Entry, err error) {
	if c.file == nil {
		err = io.EOF
		return
	}

	if len(c.entries) == 0 {
		c.entries, err = c.file.Readdir(100)
		if err != nil {
			c.cancel()
			c.file = nil
			if errors.Is(err, io.EOF) {
				err = nil
			}
		}
	}

	if len(c.entries) == 0 {
		return
	}

	entry := c.entries[0]
	c.entries = c.entries[1:]
	ret = &Entry{
		Key:      entry.Name(),
		IsPrefix: entry.IsDir(),
	}
	return
}

func (*fileFSCursor) cleanup(f *os.File, ctx context.Context) {
	<-ctx.Done()
	f.Close()
}

func (f FileFS) Remove(ctx context.Context, path string) error {
	return os.Remove(filepath.Join(f.path, path))
}

func (f FileFS) Exists(ctx context.Context, path string) (ok bool, err error) {
	_, err = os.Stat(filepath.Join(f.path, path))
	ok = err == nil
	if os.IsNotExist(err) {
		err = nil
	}
	return
}
