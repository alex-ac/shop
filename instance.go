package shop

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
)

type Instance struct {
	ApiVersion string        `json:"api_version"`
	Package    string        `json:"package"`
	Id         string        `json:"id"`
	UploadedAt UnixTimestamp `json:"uploaded_at"`
	UpdatedAt  UnixTimestamp `json:"updated_at"`
}

func NewInstance(pkg, id string) (instance Instance, err error) {
	switch {
	case !IsValidPackageName(pkg):
		err = fmt.Errorf("%w: %s", ErrInvalidPackageName, pkg)
	case !IsValidInstanceId(id):
		err = fmt.Errorf("%w: %s", ErrInvalidInstanceId, id)
	default:
		instance = Instance{
			ApiVersion: LatestVersion,
			Package:    pkg,
			Id:         id,
			UploadedAt: UnixTimestamp{time.Now()},
			UpdatedAt:  UnixTimestamp{time.Now()},
		}
	}
	return
}

func IsValidInstanceId(id string) bool {
	if len(id) != RegistryPackageInstanceIdLen {
		return false
	}
	for _, c := range id {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

type TeeWriter []io.Writer

type teeWriterError struct {
	error
	n int
}

func (w TeeWriter) Write(data []byte) (n int, err error) {
	var group multierror.Group
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, writer := range w {
		func(ctx context.Context, w io.Writer, data []byte) {
			group.Go(func() (err error) {
				var n int
				for len(data) > 0 && err == nil && ctx.Err() == nil {
					n, err = w.Write(data)
					data = data[n:]
				}
				if err == nil && ctx.Err() != nil {
					err = ctx.Err()
				}
				if err != nil {
					err = teeWriterError{error: err, n: n}
					cancel()
				}
				return
			})
		}(ctx, writer, data)
	}

	errors := group.Wait()
	n = len(data)

	for _, err := range errors.WrappedErrors() {
		if werr, ok := err.(teeWriterError); ok {
			n = min(n, werr.n)
		}
	}

	err = errors.ErrorOrNil()

	return
}

type stripOwnerFS struct {
	fs.FS
}

func (f stripOwnerFS) Open(path string) (file fs.File, err error) {
	file, err = f.FS.Open(path)
	if err == nil {
		file = stripOwnerFile{file.(fs.ReadDirFile)}
	}
	return
}

type stripOwnerDirEntry struct {
	fs.DirEntry
}

func (d stripOwnerDirEntry) Info() (info fs.FileInfo, err error) {
	info, err = d.DirEntry.Info()
	if err == nil {
		info = stripOwnerFileInfo{info}
	}
	return
}

type stripOwnerFile struct {
	fs.ReadDirFile
}

func (f stripOwnerFile) ReadDir(n int) (entries []fs.DirEntry, err error) {
	entries, err = f.ReadDirFile.ReadDir(n)
	for i, entry := range entries {
		entries[i] = stripOwnerDirEntry{entry}
	}
	return
}

func (f stripOwnerFile) Stat() (info fs.FileInfo, err error) {
	info, err = f.ReadDirFile.Stat()
	if err == nil {
		info = stripOwnerFileInfo{info}
	}
	return
}

type stripOwnerFileInfo struct {
	fs.FileInfo
}

// The only way to get uid/gid is to look into os-dependent return value of Sys.
// By returning nil, we ensure that it's impossible to find out uid/gid
// therefore in archive they are always 0.
func (fi stripOwnerFileInfo) Sys() any {
	return nil
}

func MakeArchive(dst io.Writer, fs fs.FS) (id string, err error) {
	fs = stripOwnerFS{fs}

	h := sha1.New()

	tee := TeeWriter{dst, h}
	compressor := gzip.NewWriter(tee)
	archive := tar.NewWriter(compressor)

	err = archive.AddFS(fs)
	if err != nil {
		return
	}
	archive.Close()
	if err != nil {
		return
	}
	compressor.Close()
	if err != nil {
		return
	}

	id = hex.EncodeToString(h.Sum(nil))
	return
}
