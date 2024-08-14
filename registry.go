package shop

import (
	"context"
	"encoding/json"
	"time"
)

const (
	RegistryManifestKey = "shop.json"
)

type RegistryManifest struct {
	ApiVersion string                        `json:"api_version"`
	Name       string                        `json:"name"`
	RootRepo   RepositoryManifest            `json:"root_repo"`
	Repos      map[string]RepositoryManifest `json:"repos"`
	UpdatedAt  UnixTimestamp                 `json:"updated_at"`
}

type RepositoryManifest struct {
	URL         string           `json:"url"`
	S3          S3BucketManifest `json:"s3,omitempty"`
	ReadOnlyURL string           `json:"readonly_url,omitempty"`
	UpdatedAt   UnixTimestamp    `json:"updated_at"`
}

type S3BucketManifest struct {
	Region      string `json:"region"`
	EndpointURL string `json:"endpoint_url"`
}

type Package struct {
	ApiVersion string `json:"api_version"`
	Name       string `json:"name"`
	UpdatedAt  string `json:"package"`
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
	CAS        string        `json:"cas"`
	Tags       []InstanceTag `json:"tags"`
	UploadedAt UnixTimestamp `json:"uploaded_at"`
	UpdatedAt  UnixTimestamp `json:"updated_at"`
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
	Value      string        `json:"key"`
	Id         string        `json:"key"`
	UpdatedAt  UnixTimestamp `json:"updated_at"`
}

type Cursor[T any] interface {
	HasMore() bool
	GetNext(context.Context) (T, error)
}

type RegistryClient interface {
	GetConfig() RegistryConfig

	GetManifest(ctx context.Context) (*RegistryManifest, error)
	PutManifest(context.Context, RegistryManifest) error

	GetPackage(ctx context.Context, name string) (*Package, error)
	ListPackages(ctx context.Context, prefix string) Cursor[Package]
	PutPackage(ctx context.Context, pkg Package) error

	ListPackageInstances(ctx context.Context, name string) Cursor[Instance]
	GetPackageInstanceInfo(ctx context.Context, name, id string) (*Instance, error)
	PutPackageInstanceInfo(ctx context.Context, instance Instance) error
	DeletePackageInstanceInfo(ctx context.Context, instance Instance) error

	ListPackageReferences(ctx context.Context, name string) Cursor[Reference]
	GetPackageReference(ctx context.Context, name string) (*Reference, error)
	PutPackageReference(ctx context.Context, ref Reference) error
	DeletePackageReference(ctx context.Context, ref Reference) error

	ListPackageTags(ctx context.Context, pkg Package) Cursor[Tag]
	ListPackageInstancesByTag(ctx context.Context, tag Tag) error
	PutPackageInstanceTag(ctx context.Context, tag Tag) error
	DeletePackageInsntanceTag(ctx context.Context, tag Tag) error
}

type RegistryClientImpl struct {
	config         RegistryConfig
	rootRepository Repository
	repositories   map[string]Repository
}

func (c *RegistryClientImpl) GetConfig() RegistryConfig {
	return c.config
}

func (c *RegistryClientImpl) GetManifest(ctx context.Context) (manifest *RegistryManifest, err error) {
	manifest = &RegistryManifest{}
	err = c.rootRepository.GetJSON(ctx, RegistryManifestKey, manifest)
	if err != nil {
		manifest = nil
	}
	return
}

func (*RegistryClientImpl) PutManifest(context.Context, RegistryManifest) error {
	return nil
}

func (*RegistryClientImpl) GetPackage(ctx context.Context, name string) (*Package, error) {
	return nil, nil
}

func (*RegistryClientImpl) ListPackages(ctx context.Context, prefix string) Cursor[Package] {
	return nil
}

func (*RegistryClientImpl) PutPackage(ctx context.Context, pkg Package) error {
	return nil
}

func (*RegistryClientImpl) ListPackageInstances(ctx context.Context, name string) Cursor[Instance] {
	return nil
}

func (*RegistryClientImpl) GetPackageInstanceInfo(ctx context.Context, name, id string) (*Instance, error) {
	return nil, nil
}

func (*RegistryClientImpl) PutPackageInstanceInfo(ctx context.Context, instance Instance) error {
	return nil
}

func (*RegistryClientImpl) DeletePackageInstanceInfo(ctx context.Context, instance Instance) error {
	return nil
}

func (*RegistryClientImpl) ListPackageReferences(ctx context.Context, name string) Cursor[Reference] {
	return nil
}

func (*RegistryClientImpl) GetPackageReference(ctx context.Context, name string) (*Reference, error) {
	return nil, nil
}

func (*RegistryClientImpl) PutPackageReference(ctx context.Context, ref Reference) error {
	return nil
}

func (*RegistryClientImpl) DeletePackageReference(ctx context.Context, ref Reference) error {
	return nil
}

func (*RegistryClientImpl) ListPackageTags(ctx context.Context, pkg Package) Cursor[Tag] {
	return nil
}

func (*RegistryClientImpl) ListPackageInstancesByTag(ctx context.Context, tag Tag) error {
	return nil
}

func (*RegistryClientImpl) PutPackageInstanceTag(ctx context.Context, tag Tag) error {
	return nil
}

func (*RegistryClientImpl) DeletePackageInsntanceTag(ctx context.Context, tag Tag) error {
	return nil
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
		config:         cfg,
		rootRepository: repository,
	}

	return registryClient, nil
}
