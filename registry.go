package shop

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
)

const (
	LatestVersion                      = "1.0.0"
	RegistryManifestKey                = "shop.json"
	RegistryPackagesPrefix             = "/packages/"
	RegistryPackageManifestKey         = "package.json"
	RegistryPackageReferencesPrefix    = "/refs/"
	RegistryPackageInstancesPrefix     = "/instances/"
	RegistryPackageInstanceManifestKey = "instance.json"
	RegistryPackageTagsPrefix          = "/tags/"
	RegistryPackageInstanceTagsPrefix  = "/tags/"
	RegistryPackageInstanceIdLen       = sha1.Size * 2
	RegistryCASPrefix                  = "/cas/"
	RegistryCASArchiveExtension        = ".tgz"
)

var (
	ErrUnimplemented             = errors.New("Unimplemented")
	ErrRegistryAdminIsNotAllowed = errors.New("Admin action on the registry is not enabled in configuration")
	ErrRegistryWriteIsNotAllowed = errors.New("Write action on the registry is not enabled in configuration")
	ErrUnknownRepo               = errors.New("Registry does not have repo")
)

type RegistryManifest struct {
	ApiVersion string                        `json:"api_version"`
	Name       string                        `json:"name"`
	RootRepo   RepositoryManifest            `json:"root_repo"`
	Repos      map[string]RepositoryManifest `json:"repos"`
	UpdatedAt  UnixTimestamp                 `json:"updated_at"`
}

type RepositoryManifest struct {
	ApiVersion  string        `json:"api_version"`
	URL         string        `json:"url"`
	Name        string        `json:"name"`
	ReadOnlyURL string        `json:"readonly_url,omitempty"`
	UpdatedAt   UnixTimestamp `json:"updated_at"`
}

type S3BucketManifest struct {
	Region      string `json:"region"`
	EndpointURL string `json:"endpoint_url"`
}

type Package struct {
	ApiVersion  string        `json:"api_version"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Repo        string        `json:"repo,omitempty"`
	UpdatedAt   UnixTimestamp `json:"package"`
}

type PackageOrPrefix struct {
	Package *Package
	Prefix  string
}

type UnixTimestamp struct {
	time.Time
}

func (ut UnixTimestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(ut.Time.Unix())
}

func (ut *UnixTimestamp) UnmarshalJSON(d []byte) (err error) {
	var v int64
	err = json.Unmarshal(d, &v)
	if err == nil {
		ut.Time = time.Unix(v, 0)
	}
	return
}

type Instance struct {
	ApiVersion string        `json:"api_version"`
	Package    string        `json:"package"`
	Id         string        `json:"id"`
	UploadedAt UnixTimestamp `json:"uploaded_at"`
	UpdatedAt  UnixTimestamp `json:"updated_at"`
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

type InstanceTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Reference struct {
	ApiVersion string        `json:"api_version"`
	Package    string        `json:"package"`
	Name       string        `json:"name"`
	Id         string        `json:"id"`
	UpdatedAt  UnixTimestamp `json:"updated_at"`
}

type Tag struct {
	ApiVersion string        `json:"api_version"`
	Package    string        `json:"package"`
	Key        string        `json:"key"`
	Value      string        `json:"value"`
	Id         string        `json:"id"`
	UpdatedAt  UnixTimestamp `json:"updated_at"`
}

type PackageTag struct {
	Package string
	Key     string
}

func IsValidTagName(v string) bool {
	// [A-Za-z][A-Za-z0-9._-]*[A-Za-z0-9]?
	if v == "" {
		return false
	}

	for i, r := range v {
		ok := (r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(i != 0 && (r >= '0' && r <= '9')) ||
			(i != 0 && i != len(v)-1 && (r == '.' || r == '_' || r == '-'))

		if !ok {
			return false
		}
	}

	return true
}

func IsValidTagValue(v string) bool {
	// [A-Za-z0-9_@-][A-Za-z0-9._@-]*[A-Za-z0-9_@-]?
	if v == "" {
		return false
	}

	for i, r := range v {
		ok := (r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			(r == '_' && r == '-' && r == '@') ||
			(i != 0 && i != len(v)-1 && r == '.')

		if !ok {
			return false
		}
	}

	return true
}

func IsValidRefName(v string) bool {
	// refs & tags has the same name format.
	return IsValidTagName(v)
}

type PackageTagValue struct {
	PackageTag
	Value string
}

type RegistryClient interface {
	GetConfig() RegistryConfig

	Initialize(ctx context.Context, name string) error

	GetManifest(ctx context.Context) (*RegistryManifest, error)
	PutManifest(context.Context, RegistryManifest) error

	GetPackage(ctx context.Context, name string) (*Package, error)
	ListPackages(ctx context.Context, prefix string) Cursor[PackageOrPrefix]
	PutPackage(ctx context.Context, pkg Package) error

	UploadPackageInstance(ctx context.Context, name string, reader io.ReadSeekCloser) (*Instance, error)
	ListPackageInstances(ctx context.Context, name string) Cursor[Instance]
	GetPackageInstanceInfo(ctx context.Context, name, id string) (*Instance, error)
	PutPackageInstanceInfo(ctx context.Context, instance Instance) error
	DeletePackageInstanceInfo(ctx context.Context, instance Instance) error
	ListPackageInstanceTags(ctx context.Context, instance Instance) Cursor[Tag]

	ListPackageReferences(ctx context.Context, name string) Cursor[Reference]
	GetPackageReference(ctx context.Context, pkg, name string) (*Reference, error)
	PutPackageReference(ctx context.Context, ref Reference) error
	DeletePackageReference(ctx context.Context, ref Reference) error

	ListPackageTags(ctx context.Context, names string) Cursor[PackageTag]
	ListPackageTagValues(ctx context.Context, tag PackageTag) Cursor[PackageTagValue]
	ListPackageInstancesByTag(ctx context.Context, tag PackageTagValue) Cursor[Tag]
	PutPackageInstanceTag(ctx context.Context, tag Tag) error
	DeletePackageInstanceTag(ctx context.Context, tag Tag) error
}

type RegistryClientImpl struct {
	cfg            RegistryConfig
	rootRepository Repository
	repositories   map[string]Repository
}

func (c *RegistryClientImpl) GetConfig() RegistryConfig {
	return c.cfg
}

func (c *RegistryClientImpl) GetManifest(ctx context.Context) (manifest *RegistryManifest, err error) {
	manifest = &RegistryManifest{}
	err = c.rootRepository.GetJSON(ctx, RegistryManifestKey, manifest)
	if err != nil {
		manifest = nil
	}
	return
}

func (c *RegistryClientImpl) Initialize(ctx context.Context, name string) error {
	if !c.cfg.Admin {
		return fmt.Errorf("%w: Initialize", ErrRegistryAdminIsNotAllowed)
	}

	repoManifest, err := c.rootRepository.GetManifest(ctx)
	if err != nil {
		return err
	}

	registryManifest := RegistryManifest{
		ApiVersion: LatestVersion,
		Name:       name,
		RootRepo:   repoManifest,
	}

	err = c.PutManifest(ctx, registryManifest)
	if err != nil {
		return err
	}

	err = c.rootRepository.EnsurePrefix(ctx, RegistryPackagesPrefix)
	if err != nil {
		return err
	}

	return err
}

func (c *RegistryClientImpl) PutManifest(ctx context.Context, manifest RegistryManifest) error {
	manifest.UpdatedAt = UnixTimestamp{time.Now()}
	if !c.cfg.Admin {
		return fmt.Errorf("%w: PutManifest", ErrRegistryAdminIsNotAllowed)
	}
	return c.rootRepository.PutJSON(ctx, RegistryManifestKey, manifest)
}

func (c *RegistryClientImpl) isPackage(ctx context.Context, name string) (bool, error) {
	key := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageManifestKey)
	return c.rootRepository.ResourceExists(ctx, key)
}

func (c *RegistryClientImpl) GetPackage(ctx context.Context, name string) (manifest *Package, err error) {
	manifest = &Package{}
	key := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageManifestKey)
	err = c.rootRepository.GetJSON(ctx, key, manifest)
	if err != nil {
		manifest = nil
	}
	return
}

type registryListPackagesCursor struct {
	client *RegistryClientImpl
	prefix string
	cursor Cursor[Entry]
}

func (c registryListPackagesCursor) GetNext(ctx context.Context) (item *PackageOrPrefix, err error) {
	for {
		var entry *Entry
		entry, err = c.cursor.GetNext(ctx)
		if err != nil || entry == nil {
			break
		}

		if !entry.IsPrefix {
			continue
		}

		key := filepath.Join(c.prefix, entry.Key)
		var isPackage bool
		isPackage, err = c.client.isPackage(ctx, key)
		if err != nil {
			break
		}

		if isPackage {
			var pkg *Package
			pkg, err = c.client.GetPackage(ctx, key)
			if err != nil {
				break
			}
			item = &PackageOrPrefix{
				Package: pkg,
			}
		} else {
			item = &PackageOrPrefix{
				Prefix: key,
			}
		}

		break
	}

	return
}

func (c *RegistryClientImpl) ListPackages(ctx context.Context, prefix string) Cursor[PackageOrPrefix] {
	isPackage, err := c.isPackage(ctx, prefix)
	if err != nil {
		return NewErrorCursor[PackageOrPrefix](err)
	}
	if isPackage {
		pkg, err := c.GetPackage(ctx, prefix)
		if err != nil {
			return NewErrorCursor[PackageOrPrefix](err)
		}

		return NewSliceCursor([]PackageOrPrefix{
			PackageOrPrefix{
				Package: pkg,
			},
		})
	}
	return registryListPackagesCursor{
		client: c,
		prefix: prefix,
		cursor: c.rootRepository.List(ctx, filepath.Join(RegistryPackagesPrefix, prefix)),
	}
}

func (c *RegistryClientImpl) PutPackage(ctx context.Context, pkg Package) error {
	pkg.ApiVersion = LatestVersion
	pkg.UpdatedAt = UnixTimestamp{time.Now()}
	prefix := filepath.Join(RegistryPackagesPrefix, pkg.Name)
	key := filepath.Join(prefix, RegistryPackageManifestKey)
	if !c.cfg.Admin {
		return fmt.Errorf("%w: PutPackage: %s", ErrRegistryAdminIsNotAllowed, pkg.Name)
	}

	err := c.rootRepository.EnsurePrefix(ctx, prefix)
	if err != nil {
		return err
	}

	err = c.rootRepository.PutJSON(ctx, key, pkg)
	if err != nil {
		return err
	}

	err = c.rootRepository.EnsurePrefix(ctx, filepath.Join(prefix, RegistryPackageInstancesPrefix))
	if err != nil {
		return err
	}

	err = c.rootRepository.EnsurePrefix(ctx, filepath.Join(prefix, RegistryPackageReferencesPrefix))
	if err != nil {
		return err
	}

	err = c.rootRepository.EnsurePrefix(ctx, filepath.Join(prefix, RegistryPackageTagsPrefix))
	if err != nil {
		return err
	}

	return nil
}

type registryListPackageInstancesCursor struct {
	cursor Cursor[Entry]
	pkg    string
	client *RegistryClientImpl
}

func (c registryListPackageInstancesCursor) GetNext(ctx context.Context) (instance *Instance, err error) {
	for {
		var entry *Entry
		entry, err = c.cursor.GetNext(ctx)
		if err != nil || entry == nil {
			break
		}

		if !entry.IsPrefix {
			err = nil
			continue
		}

		if !IsValidInstanceId(entry.Key) {
			continue
		}

		instance, err = c.client.GetPackageInstanceInfo(ctx, c.pkg, entry.Key)
		break
	}

	return
}

func (c *RegistryClientImpl) ListPackageInstances(ctx context.Context, name string) Cursor[Instance] {
	prefix := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageInstancesPrefix)
	return registryListPackageInstancesCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		pkg:    name,
		client: c,
	}
}

func (c *RegistryClientImpl) GetPackageInstanceInfo(ctx context.Context, name, id string) (instance *Instance, err error) {
	key := filepath.Join(RegistryPackagesPrefix, name, RegistryPackagesPrefix, id, RegistryPackageInstanceManifestKey)
	instance = &Instance{}
	err = c.rootRepository.GetJSON(ctx, key, instance)
	if err != nil {
		instance = nil
	}
	return
}

func (c *RegistryClientImpl) UploadPackageInstance(ctx context.Context, name string, reader io.ReadSeekCloser) (*Instance, error) {
	pkg, err := c.GetPackage(ctx, name)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if pkg.Repo != "" {
		var ok bool
		repo, ok = c.repositories[pkg.Repo]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownRepo, pkg.Repo)
		}
	} else {
		repo = c.rootRepository
	}

	_, err = reader.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	h := sha1.New()
	_, err = io.Copy(h, reader)
	if err != nil {
		return nil, err
	}

	id := hex.EncodeToString(h.Sum(nil))
	instance := &Instance{
		ApiVersion: LatestVersion,
		Package:    name,
		Id:         id,
	}

	_, err = reader.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	err = repo.EnsurePrefix(ctx, RegistryCASPrefix)
	if err != nil {
		return nil, err
	}

	casKey := filepath.Join(RegistryCASPrefix, id+RegistryCASArchiveExtension)
	err = repo.Put(ctx, casKey, reader)
	if err != nil {
		return nil, err
	}

	instance.UploadedAt = UnixTimestamp{time.Now()}
	return instance, nil
}

func (c *RegistryClientImpl) PutPackageInstanceInfo(ctx context.Context, instance Instance) error {
	key := filepath.Join(RegistryPackagesPrefix, instance.Package, RegistryPackageInstancesPrefix, instance.Id, RegistryPackageInstanceManifestKey)
	prefix := filepath.Dir(key)
	tagsPrefix := filepath.Join(prefix, RegistryPackageInstanceTagsPrefix)
	if !c.cfg.Write {
		return fmt.Errorf("%w: %s / %s", ErrRegistryWriteIsNotAllowed, instance.Package, instance.Id)
	}

	if err := c.rootRepository.EnsurePrefix(ctx, prefix); err != nil {
		return err
	}

	instance.UpdatedAt = UnixTimestamp{time.Now()}

	return multierror.Append(
		c.rootRepository.PutJSON(ctx, key, instance),
		c.rootRepository.EnsurePrefix(ctx, tagsPrefix),
	).ErrorOrNil()
}

func (c *RegistryClientImpl) DeletePackageInstanceInfo(ctx context.Context, instance Instance) error {
	key := filepath.Join(RegistryPackagesPrefix, instance.Package, RegistryPackagesPrefix, instance.Id)
	if !c.cfg.Admin {
		return fmt.Errorf("%w: %s / %s", ErrRegistryAdminIsNotAllowed, instance.Package, instance.Id)
	}
	if !c.cfg.Write {
		return fmt.Errorf("%w: %s / %s", ErrRegistryWriteIsNotAllowed, instance.Package, instance.Id)
	}
	return c.rootRepository.Delete(ctx, key)
}

type registryInstanceTagsCursor struct {
	cursor   Cursor[Entry]
	instance Instance
	client   *RegistryClientImpl
}

func (c registryInstanceTagsCursor) GetNext(ctx context.Context) (tag *Tag, err error) {
	for {
		var entry *Entry
		entry, err = c.cursor.GetNext(ctx)
		if err != nil || entry == nil {
			break
		}

		if entry.IsPrefix {
			continue
		}

		key := filepath.Join(RegistryPackagesPrefix, c.instance.Package, RegistryPackageInstancesPrefix, c.instance.Id, RegistryPackageInstanceTagsPrefix, entry.Key)
		tag = &Tag{}
		err = c.client.rootRepository.GetJSON(ctx, key, tag)
		if err != nil {
			tag = nil
		}
		break
	}
	return
}

func (c *RegistryClientImpl) ListPackageInstanceTags(ctx context.Context, instance Instance) Cursor[Tag] {
	prefix := filepath.Join(RegistryPackagesPrefix, instance.Package, RegistryPackageInstancesPrefix, instance.Id, RegistryPackageInstanceTagsPrefix)
	return registryInstanceTagsCursor{
		cursor:   c.rootRepository.List(ctx, prefix),
		instance: instance,
		client:   c,
	}
}

type registryListRefsCursor struct {
	cursor Cursor[Entry]
	pkg    string
	client *RegistryClientImpl
}

func (c registryListRefsCursor) GetNext(ctx context.Context) (ref *Reference, err error) {
	for {
		var entry *Entry
		entry, err = c.cursor.GetNext(ctx)
		if err != nil || entry == nil {
			break
		}

		if entry.IsPrefix {
			err = nil
			continue
		}

		ref, err = c.client.GetPackageReference(ctx, c.pkg, entry.Key)
		break
	}
	return
}

func (c *RegistryClientImpl) ListPackageReferences(ctx context.Context, name string) Cursor[Reference] {
	prefix := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageReferencesPrefix)
	return registryListRefsCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		pkg:    name,
		client: c,
	}
}

func (c *RegistryClientImpl) GetPackageReference(ctx context.Context, pkg, name string) (ref *Reference, err error) {
	key := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageReferencesPrefix, name)
	ref = &Reference{}
	err = c.rootRepository.GetJSON(ctx, key, ref)
	if err != nil {
		ref = nil
	}
	return
}

func (c *RegistryClientImpl) PutPackageReference(ctx context.Context, ref Reference) error {
	key := filepath.Join(RegistryPackagesPrefix, ref.Package, RegistryPackageReferencesPrefix, ref.Name)
	if !c.cfg.Write {
		return fmt.Errorf("%w: %s / %s", ErrRegistryWriteIsNotAllowed, ref.Package, ref.Name)
	}
	return c.rootRepository.PutJSON(ctx, key, ref)
}

func (c *RegistryClientImpl) DeletePackageReference(ctx context.Context, ref Reference) error {
	key := filepath.Join(RegistryPackagesPrefix, ref.Package, RegistryPackageReferencesPrefix, ref.Name)
	if !c.cfg.Admin {
		return fmt.Errorf("%w: %s / %s", ErrRegistryAdminIsNotAllowed, ref.Package, ref.Name)
	}
	if !c.cfg.Write {
		return fmt.Errorf("%w: %s / %s", ErrRegistryWriteIsNotAllowed, ref.Package, ref.Name)
	}
	return c.rootRepository.Delete(ctx, key)
}

type registryListTagsCursor struct {
	cursor Cursor[Entry]
	pkg    string
}

func (c registryListTagsCursor) GetNext(ctx context.Context) (tag *PackageTag, err error) {
	for {
		var entry *Entry
		entry, err = c.cursor.GetNext(ctx)
		if err != nil || entry == nil {
			break
		}

		if !entry.IsPrefix {
			continue
		}

		tag = &PackageTag{
			Package: c.pkg,
			Key:     entry.Key,
		}
		break
	}

	return
}

func (c *RegistryClientImpl) ListPackageTags(ctx context.Context, name string) Cursor[PackageTag] {
	prefix := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageTagsPrefix)
	return registryListTagsCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		pkg:    name,
	}
}

type registryListPackageTagValuesCursor struct {
	cursor Cursor[Entry]
	tag    PackageTag
}

func (c registryListPackageTagValuesCursor) GetNext(ctx context.Context) (tag *PackageTagValue, err error) {
	for {
		var entry *Entry
		entry, err = c.cursor.GetNext(ctx)
		if err != nil || entry == nil {
			break
		}

		if !entry.IsPrefix {
			continue
		}

		tag = &PackageTagValue{
			PackageTag: c.tag,
			Value:      entry.Key,
		}
		break
	}
	return
}

func (c *RegistryClientImpl) ListPackageTagValues(ctx context.Context, tag PackageTag) Cursor[PackageTagValue] {
	prefix := filepath.Join(RegistryPackagesPrefix, tag.Package, RegistryPackageTagsPrefix, tag.Key)
	return registryListPackageTagValuesCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		tag:    tag,
	}
}

type registryListPackageInstancesByTagCursor struct {
	cursor Cursor[Entry]
	tag    PackageTagValue
	client *RegistryClientImpl
}

func (c registryListPackageInstancesByTagCursor) GetNext(ctx context.Context) (tag *Tag, err error) {
	for {
		var entry *Entry
		entry, err = c.cursor.GetNext(ctx)
		if err != nil || entry == nil {
			break
		}

		if entry.IsPrefix || !IsValidInstanceId(entry.Key) {
			continue
		}

		key := filepath.Join(RegistryPackagesPrefix, c.tag.Package, c.tag.Key, c.tag.Value, entry.Key)

		tag = &Tag{}
		err = c.client.rootRepository.GetJSON(ctx, key, tag)
		if err != nil {
			tag = nil
		}
		break
	}
	return
}

func (c *RegistryClientImpl) ListPackageInstancesByTag(ctx context.Context, tag PackageTagValue) Cursor[Tag] {
	prefix := filepath.Join(RegistryPackagesPrefix, tag.Package, RegistryPackageTagsPrefix, tag.Key, tag.Value)
	return registryListPackageInstancesByTagCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		tag:    tag,
		client: c,
	}
}

func (c *RegistryClientImpl) PutPackageInstanceTag(ctx context.Context, tag Tag) error {
	tag.ApiVersion = LatestVersion
	tag.UpdatedAt = UnixTimestamp{time.Now()}

	if !c.cfg.Write {
		return fmt.Errorf("%w: %s/%s:%s -> %s", ErrRegistryWriteIsNotAllowed, tag.Package, tag.Key, tag.Value, tag.Id)
	}

	key1 := filepath.Join(RegistryPackagesPrefix, tag.Package, RegistryPackageTagsPrefix, tag.Key, tag.Value, tag.Id)
	prefix1 := filepath.Dir(key1)
	key2 := filepath.Join(RegistryPackagesPrefix, tag.Package, RegistryPackageInstancesPrefix, tag.Id, RegistryPackageInstanceTagsPrefix, tag.Key, tag.Value)
	prefix2 := filepath.Dir(key2)

	if err := multierror.Append(
		c.rootRepository.EnsurePrefix(ctx, prefix1),
		c.rootRepository.EnsurePrefix(ctx, prefix2),
	).ErrorOrNil(); err != nil {
		return err
	}

	return multierror.Append(
		c.rootRepository.PutJSON(ctx, key1, tag),
		c.rootRepository.PutJSON(ctx, key2, tag),
	).ErrorOrNil()
}

func (c *RegistryClientImpl) DeletePackageInstanceTag(ctx context.Context, tag Tag) error {
	if !c.cfg.Admin {
		return fmt.Errorf("%w: Delete Tag: %s/%s/%s -> %s", ErrRegistryAdminIsNotAllowed, tag.Package, tag.Key, tag.Value, tag.Id)
	}

	key1 := filepath.Join(RegistryPackagesPrefix, tag.Package, RegistryPackageTagsPrefix, tag.Key, tag.Value, tag.Id)
	key2 := filepath.Join(RegistryPackagesPrefix, tag.Package, RegistryPackageInstancesPrefix, tag.Id, RegistryPackageInstanceTagsPrefix, tag.Key)

	return multierror.Append(
		c.rootRepository.Delete(ctx, key1),
		c.rootRepository.Delete(ctx, key2),
	).ErrorOrNil()
}

var _ RegistryClient = (*RegistryClientImpl)(nil)

func NewRegistryClient(ctx context.Context, cfg RegistryConfig) (RegistryClient, error) {

	if cfg.RootRepo.URL == "" {
		cfg.RootRepo.URL = cfg.URL
	}

	repository, err := NewRepository(ctx, cfg.RootRepo)
	if err != nil {
		return nil, err
	}

	cfg.RootRepo = repository.GetConfig()

	var registryClient RegistryClient = &RegistryClientImpl{
		cfg:            cfg,
		rootRepository: repository,
	}

	return registryClient, nil
}
