package shop

import (
	"context"
	"crypto/sha1"
	"io"
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

type RegistryManifest struct {
	ApiVersion string                        `json:"api_version"`
	Name       string                        `json:"name"`
	RootRepo   RepositoryManifest            `json:"root_repo"`
	Repos      map[string]RepositoryManifest `json:"repos"`
	UpdatedAt  UnixTimestamp                 `json:"updated_at"`
}

type PackageOrPrefix struct {
	Package *Package
	Prefix  string
}

type Registry interface {
	GetConfig() RegistryConfig

	Initialize(ctx context.Context, name string) error

	GetManifest(ctx context.Context) (*RegistryManifest, error)
	PutManifest(context.Context, RegistryManifest) error

	GetPackage(ctx context.Context, name string) (*Package, error)
	ListPackages(ctx context.Context, prefix string) Cursor[PackageOrPrefix]
	PutPackage(ctx context.Context, pkg Package) error

	UploadPackageInstance(ctx context.Context, name, id string, reader io.Reader) (*Instance, error)
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
