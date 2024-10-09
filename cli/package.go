package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/alex-ac/shop"
	"github.com/spf13/cobra"
)

var (
	ErrRegistryDoesNotExist = errors.New("Registry does not exist")
)

type PackageCommand struct {
	Arguments    *GlobalArguments
	RegistryName string
	Cfg          shop.Config
}

func NewPackageCommand(args *GlobalArguments) *cobra.Command {
	c := &PackageCommand{
		Arguments: args,
	}

	cmd := &cobra.Command{
		Use:   "package [-r registry]",
		Short: "Manage packages.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			c.Cfg, err = c.Arguments.LoadConfig()
			if err != nil {
				return
			}

			if c.RegistryName == "" {
				c.RegistryName = c.Cfg.DefaultRegistry
			}

			if c.RegistryName == "" {
				c.RegistryName = shop.DefaultRegistryName
			}

			if _, ok := c.Cfg.Registries[c.RegistryName]; !ok {
				err = fmt.Errorf("%w: %s", ErrRegistryDoesNotExist, c.RegistryName)
			}
			return
		},
	}

	cmd.AddCommand(
		NewPackageListCommand(c),
		NewPackageAddCommand(c),
		NewPackageUploadCommand(c),
	)

	cmd.PersistentFlags().StringVarP(&c.RegistryName, "registry", "r", "", "Registry name.")

	return cmd
}

type PackageListCommand struct {
	*PackageCommand
}

func NewPackageListCommand(parent *PackageCommand) *cobra.Command {
	c := &PackageListCommand{
		PackageCommand: parent,
	}

	cmd := &cobra.Command{
		Use:   "ls [prefix]",
		Short: "List packages in registry.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prefix := ""
			if len(args) > 0 {
				prefix = args[0]
			}
			return c.Run(cmd.Context(), prefix)
		},
	}

	return cmd
}

func (c *PackageListCommand) Run(ctx context.Context, prefix string) error {
	registryConfig := c.Cfg.Registries[c.RegistryName]

	registryClient, err := shop.NewRegistryClient(ctx, registryConfig)
	if err != nil {
		return err
	}

	var output []PackageListOutputItem
	cursor := registryClient.ListPackages(ctx, prefix)
	for {
		pkg, err := cursor.GetNext(ctx)
		if err != nil {
			return err
		}
		if pkg == nil {
			break
		}

		output = append(output, PackageListOutputItem{pkg})
	}

	encoder := c.Arguments.OutputFormat.CreateEncoder(os.Stdout)
	return encoder.Encode(output)
}

type PackageListOutputItem struct {
	*shop.PackageOrPrefix
}

func (i PackageListOutputItem) IntoText() (text []byte, err error) {
	if i.Package != nil {
		text = append(text, i.Package.Name...)
		if i.Package.Description != "" {
			text = append(text, "\t"...)
			text = append(text, i.Package.Description...)
		}
		if i.Package.Repo != "" {
			text = append(text, "\trepo="...)
			text = append(text, i.Package.Repo...)
		}
	} else {
		text = append(text, i.Prefix...)
		text = append(text, "/"...)
	}
	return
}

type PackageAddCommand struct {
	*PackageCommand

	Description string
	Repo        string
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

func NewPackageAddCommand(parent *PackageCommand) *cobra.Command {
	c := &PackageAddCommand{
		PackageCommand: parent,
	}

	cmd := &cobra.Command{
		Use:   "add [-d description] [-R repo] package_name",
		Short: "Add Package into registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd.Context(), args[0])
		},
	}

	cmd.PersistentFlags().StringVarP(&c.Description, "description", "d", "", "Package description text.")
	cmd.PersistentFlags().StringVarP(&c.Repo, "repo", "R", "", "Repo to use for package data")

	return cmd
}

func (c *PackageAddCommand) Run(ctx context.Context, name string) error {
	registryConfig := c.Cfg.Registries[c.RegistryName]

	registryClient, err := shop.NewRegistryClient(ctx, registryConfig)
	if err != nil {
		return err
	}

	if c.Repo != "" {
		registryManifest, err := registryClient.GetManifest(ctx)
		if err != nil {
			return err
		}

		if _, ok := registryManifest.Repos[c.Repo]; !ok {
			return fmt.Errorf("%w: %s", shop.ErrUnknownRepo, c.Repo)
		}
	}

	pkg := shop.Package{
		ApiVersion:  shop.LatestVersion,
		Name:        name,
		Description: c.Description,
		Repo:        c.Repo,
	}

	return registryClient.PutPackage(ctx, pkg)
}

type TagsMap map[string]string

func (m TagsMap) String() string {
	var items []string
	for key, value := range m {
		items = append(items, fmt.Sprintf("%s:%s", key, value))
	}
	return strings.Join(items, ",")
}

func (m TagsMap) Set(v string) error {
	for _, pair := range strings.Split(v, ",") {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 {
			return fmt.Errorf("%s must be formatted as key:value", pair)
		}
		key, value := kv[0], kv[1]

		if !shop.IsValidTagName(key) {
			return fmt.Errorf("%s is not a valid tag name", key)
		}
		if !shop.IsValidTagValue(value) {
			return fmt.Errorf("%s is not a valid tag value", value)
		}

		if oldValue, ok := m[key]; ok && oldValue != value {
			return fmt.Errorf("Conflicting values for tag %s: %s vs %s", key, oldValue, value)
		}
		m[key] = value
	}

	return nil
}

func (m TagsMap) Type() string {
	return "tag:value"
}

type RefSet map[string]struct{}

func (s RefSet) String() string {
	var items []string
	for ref, _ := range s {
		items = append(items, ref)
	}
	return strings.Join(items, ",")
}

func (s RefSet) Type() string {
	return "refs"
}

func (s RefSet) Set(v string) error {
	for _, ref := range strings.Split(v, ",") {
		if !shop.IsValidRefName(ref) {
			return fmt.Errorf("%s is not a valid ref name", ref)
		}

		s[ref] = struct{}{}
	}

	return nil
}

type PackageUploadCommand struct {
	*PackageCommand

	Tags TagsMap
	Refs RefSet
	Dir  string
}

func NewPackageUploadCommand(parent *PackageCommand) *cobra.Command {
	c := &PackageUploadCommand{
		PackageCommand: parent,
		Tags:           TagsMap{},
		Refs:           RefSet{},
	}

	cmd := &cobra.Command{
		Use:   "upload [-t tag:value...] [-R ref] package_name dir",
		Short: "Upload new instance for package.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd.Context(), args[0], args[1])
		},
	}

	cmd.PersistentFlags().VarP(c.Tags, "tag", "t", "Attach tag(s) to the instance.")
	cmd.PersistentFlags().VarP(c.Refs, "ref", "R", "Update reference to point to the instance.")

	return cmd
}

func (c *PackageUploadCommand) Run(ctx context.Context, name, dir string) error {
	registryConfig := c.Cfg.Registries[c.RegistryName]

	registryClient, err := shop.NewRegistryClient(ctx, registryConfig)
	if err != nil {
		return err
	}

	file, err := os.CreateTemp("", fmt.Sprintf("%s_*%s", strings.Replace(name, "/", "-", -1), shop.RegistryCASArchiveExtension))
	if err != nil {
		return err
	}
	defer file.Close()

	err = os.Remove(file.Name())
	if err != nil {
		return err
	}

	compressor := gzip.NewWriter(file)
	archive := tar.NewWriter(compressor)
	err = archive.AddFS(stripOwnerFS{os.DirFS(dir)})
	if err != nil {
		return err
	}

	err = archive.Close()
	if err != nil {
		return err
	}

	err = compressor.Close()
	if err != nil {
		return err
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	instance, err := registryClient.UploadPackageInstance(ctx, name, file)
	if err != nil {
		return err
	}

	err = registryClient.PutPackageInstanceInfo(ctx, *instance)
	if err != nil {
		return err
	}

	fmt.Printf("%s@%s\n", name, instance.Id)

	for key, value := range c.Tags {
		err = registryClient.PutPackageInstanceTag(ctx, shop.Tag{
			ApiVersion: shop.LatestVersion,
			Package:    name,
			Key:        key,
			Value:      value,
			Id:         instance.Id,
			UpdatedAt:  shop.UnixTimestamp{time.Now()},
		})
		if err != nil {
			return err
		}
		fmt.Printf("%s@%s:%s\n", name, key, value)
	}

	for ref, _ := range c.Refs {
		err = registryClient.PutPackageReference(ctx, shop.Reference{
			ApiVersion: shop.LatestVersion,
			Package:    name,
			Name:       ref,
			Id:         instance.Id,
			UpdatedAt:  shop.UnixTimestamp{time.Now()},
		})
		if err != nil {
			return err
		}
		fmt.Printf("%s@%s\n", name, ref)
	}

	return nil
}
