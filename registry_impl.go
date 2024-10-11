package shop

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-multierror"
)

type RegistryImpl struct {
	cfg            RegistryConfig
	rootRepository Repository
	repositories   map[string]Repository
}

func (c *RegistryImpl) GetConfig() RegistryConfig {
	return c.cfg
}

func (c *RegistryImpl) GetManifest(ctx context.Context) (manifest *RegistryManifest, err error) {
	manifest = &RegistryManifest{}
	err = c.rootRepository.GetJSON(ctx, RegistryManifestKey, manifest)
	if err != nil {
		manifest = nil
	}
	return
}

func (c *RegistryImpl) Initialize(ctx context.Context, name string) error {
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

	err = multierror.Append(
		c.rootRepository.EnsurePrefix(ctx, RegistryPackagesPrefix),
		c.rootRepository.EnsurePrefix(ctx, RegistryCASPrefix),
	).ErrorOrNil()

	return err
}

func (c *RegistryImpl) PutManifest(ctx context.Context, manifest RegistryManifest) error {
	manifest.UpdatedAt = UnixTimestamp{time.Now()}
	if !c.cfg.Admin {
		return fmt.Errorf("%w: PutManifest", ErrRegistryAdminIsNotAllowed)
	}
	return c.rootRepository.PutJSON(ctx, RegistryManifestKey, manifest)
}

func (c *RegistryImpl) isPackage(ctx context.Context, name string) (bool, error) {
	key := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageManifestKey)
	return c.rootRepository.ResourceExists(ctx, key)
}

func (c *RegistryImpl) GetPackage(ctx context.Context, name string) (manifest *Package, err error) {
	manifest = &Package{}
	key := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageManifestKey)
	err = c.rootRepository.GetJSON(ctx, key, manifest)
	if err != nil {
		manifest = nil
	}
	return
}

type registryListPackagesCursor struct {
	client *RegistryImpl
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

func (c *RegistryImpl) ListPackages(ctx context.Context, prefix string) Cursor[PackageOrPrefix] {
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

func (c *RegistryImpl) PutPackage(ctx context.Context, pkg Package) error {
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

	err = multierror.Append(
		c.rootRepository.EnsurePrefix(ctx, filepath.Join(prefix, RegistryPackageInstancesPrefix)),
		c.rootRepository.EnsurePrefix(ctx, filepath.Join(prefix, RegistryPackageReferencesPrefix)),
		c.rootRepository.EnsurePrefix(ctx, filepath.Join(prefix, RegistryPackageTagsPrefix)),
	)
	if err != nil {
		return err
	}

	return nil
}

type registryListPackageInstancesCursor struct {
	cursor Cursor[Entry]
	pkg    string
	client *RegistryImpl
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

func (c *RegistryImpl) ListPackageInstances(ctx context.Context, name string) Cursor[Instance] {
	prefix := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageInstancesPrefix)
	return registryListPackageInstancesCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		pkg:    name,
		client: c,
	}
}

func (c *RegistryImpl) GetPackageInstanceInfo(ctx context.Context, name, id string) (instance *Instance, err error) {
	key := filepath.Join(RegistryPackagesPrefix, name, RegistryPackagesPrefix, id, RegistryPackageInstanceManifestKey)
	instance = &Instance{}
	err = c.rootRepository.GetJSON(ctx, key, instance)
	if err != nil {
		instance = nil
	}
	return
}

func (c *RegistryImpl) UploadPackageInstance(ctx context.Context, name, id string, reader io.Reader) (*Instance, error) {
	if !c.cfg.Write {
		return nil, fmt.Errorf("%w: %s@%s", ErrRegistryWriteIsNotAllowed, name, id)
	}
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

	instance, err := NewInstance(name, id)
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
	return &instance, nil
}

func (c *RegistryImpl) PutPackageInstanceInfo(ctx context.Context, instance Instance) error {
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

func (c *RegistryImpl) DeletePackageInstanceInfo(ctx context.Context, instance Instance) error {
	key := filepath.Join(RegistryPackagesPrefix, instance.Package, RegistryPackagesPrefix, instance.Id)
	if !c.cfg.Admin {
		return fmt.Errorf("%w: %s / %s", ErrRegistryAdminIsNotAllowed, instance.Package, instance.Id)
	}
	return c.rootRepository.Delete(ctx, key)
}

type registryInstanceTagsCursor struct {
	cursor   Cursor[Entry]
	instance Instance
	client   *RegistryImpl
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

func (c *RegistryImpl) ListPackageInstanceTags(ctx context.Context, instance Instance) Cursor[Tag] {
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
	client *RegistryImpl
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

func (c *RegistryImpl) ListPackageReferences(ctx context.Context, name string) Cursor[Reference] {
	prefix := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageReferencesPrefix)
	return registryListRefsCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		pkg:    name,
		client: c,
	}
}

func (c *RegistryImpl) GetPackageReference(ctx context.Context, pkg, name string) (ref *Reference, err error) {
	key := filepath.Join(RegistryPackagesPrefix, name, RegistryPackageReferencesPrefix, name)
	ref = &Reference{}
	err = c.rootRepository.GetJSON(ctx, key, ref)
	if err != nil {
		ref = nil
	}
	return
}

func (c *RegistryImpl) PutPackageReference(ctx context.Context, ref Reference) error {
	key := filepath.Join(RegistryPackagesPrefix, ref.Package, RegistryPackageReferencesPrefix, ref.Name)
	if !c.cfg.Write {
		return fmt.Errorf("%w: %s / %s", ErrRegistryWriteIsNotAllowed, ref.Package, ref.Name)
	}
	return c.rootRepository.PutJSON(ctx, key, ref)
}

func (c *RegistryImpl) DeletePackageReference(ctx context.Context, ref Reference) error {
	key := filepath.Join(RegistryPackagesPrefix, ref.Package, RegistryPackageReferencesPrefix, ref.Name)
	if !c.cfg.Admin {
		return fmt.Errorf("%w: %s / %s", ErrRegistryAdminIsNotAllowed, ref.Package, ref.Name)
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

func (c *RegistryImpl) ListPackageTags(ctx context.Context, name string) Cursor[PackageTag] {
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

func (c *RegistryImpl) ListPackageTagValues(ctx context.Context, tag PackageTag) Cursor[PackageTagValue] {
	prefix := filepath.Join(RegistryPackagesPrefix, tag.Package, RegistryPackageTagsPrefix, tag.Key)
	return registryListPackageTagValuesCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		tag:    tag,
	}
}

type registryListPackageInstancesByTagCursor struct {
	cursor Cursor[Entry]
	tag    PackageTagValue
	client *RegistryImpl
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

func (c *RegistryImpl) ListPackageInstancesByTag(ctx context.Context, tag PackageTagValue) Cursor[Tag] {
	prefix := filepath.Join(RegistryPackagesPrefix, tag.Package, RegistryPackageTagsPrefix, tag.Key, tag.Value)
	return registryListPackageInstancesByTagCursor{
		cursor: c.rootRepository.List(ctx, prefix),
		tag:    tag,
		client: c,
	}
}

func (c *RegistryImpl) PutPackageInstanceTag(ctx context.Context, tag Tag) error {
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

func (c *RegistryImpl) DeletePackageInstanceTag(ctx context.Context, tag Tag) error {
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

var _ Registry = (*RegistryImpl)(nil)

func NewRegistry(ctx context.Context, cfg RegistryConfig) (Registry, error) {
	if cfg.RootRepo.URL == "" {
		cfg.RootRepo.URL = cfg.URL
	}

	repository, err := NewRepository(ctx, cfg.RootRepo)
	if err != nil {
		return nil, err
	}

	cfg.RootRepo = repository.GetConfig()

	registryClient := &RegistryImpl{
		cfg:            cfg,
		rootRepository: repository,
		repositories:   map[string]Repository{},
	}

	manifest, err := registryClient.GetManifest(ctx)
	if err != nil {
		return nil, err
	}

	for key, repoManifest := range manifest.Repos {
		repoCfg, ok := cfg.Repos[key]
		if !ok || repoCfg.URL != repoManifest.URL {
			repoCfg.URL = repoManifest.URL
			repoCfg.Admin = cfg.Admin || repoCfg.Admin
			repoCfg.Write = repoCfg.Admin || cfg.Write || repoCfg.Write
			cfg.Repos[key] = repoCfg
		}

		repo, err := NewRepository(ctx, repoCfg)
		if err != nil {
			return nil, err
		}
		registryClient.repositories[key] = repo
	}

	return registryClient, nil
}
