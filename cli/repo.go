package cli

import (
	"context"
	"net/url"

	"github.com/alex-ac/shop"
	"github.com/spf13/cobra"
)

type RepoCommand struct {
	Arguments *GlobalArguments
}

func NewRepoCommand(args *GlobalArguments) *cobra.Command {
	c := &RepoCommand{
		Arguments: args,
	}

	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories in the registry.",
	}

	cmd.AddCommand(
		NewRepoAddCommand(c),
		NewRepoInitCommand(c),
	)

	return cmd
}

type RepoAddCommand struct {
	*RepoCommand

	Registry string
	Name     string
	URL      string
}

func (c *RepoAddCommand) Run(ctx context.Context, url string) error {
	return shop.ErrUnimplemented
}

func NewRepoAddCommand(parent *RepoCommand) *cobra.Command {
	c := &RepoAddCommand{
		RepoCommand: parent,
		Registry:    shop.DefaultRegistryName,
	}

	cmd := &cobra.Command{
		Use:   "add [-n name] url",
		Short: "Add repository to the registry.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd.Context(), args[0])
		},
	}

	cmd.PersistentFlags().StringVarP(&c.Registry, "registry", "r", c.Registry, "Registry to operate on")
	cmd.PersistentFlags().StringVarP(&c.Name, "name", "n", "", "Name of the repo to override one from the manifest.")
	cmd.MarkPersistentFlagRequired("registry")

	return cmd
}

type RepoInitCommand struct {
	*RepoCommand
	Name        string
	ReadOnlyURL string
}

func NewRepoInitCommand(parent *RepoCommand) *cobra.Command {
	c := &RepoInitCommand{
		RepoCommand: parent,
	}

	cmd := &cobra.Command{
		Use:   "init -n name url",
		Short: "Initialize new repository.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd.Context(), args[0])
		},
	}

	cmd.PersistentFlags().StringVarP(&c.Name, "name", "n", "", "Name of the repository in the manifest.")
	cmd.MarkPersistentFlagRequired("name")
	cmd.PersistentFlags().StringVarP(&c.ReadOnlyURL, "ro-url", "r", "", "Read-Only URL (likely http).")

	return cmd
}

func (c *RepoInitCommand) Run(ctx context.Context, u string) error {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return err
	}

	manifest := shop.RepositoryManifest{
		ApiVersion: shop.LatestVersion,
		URL:        parsedUrl.String(),
		Name:       c.Name,
	}

	if c.ReadOnlyURL != "" {
		_, err := url.Parse(c.ReadOnlyURL)
		if err != nil {
			return err
		}
		manifest.ReadOnlyURL = c.ReadOnlyURL
	}

	config := shop.RepositoryConfig{
		URL:   parsedUrl.String(),
		Write: true,
		Admin: true,
	}

	repo, err := shop.NewRepository(ctx, config)
	if err != nil {
		return err
	}

	return repo.PutManifest(ctx, manifest)
}
