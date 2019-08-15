package generate

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
	"github.com/weaveworks/eksctl/pkg/ctl/cmdutils"
	"github.com/weaveworks/eksctl/pkg/git"
	"github.com/weaveworks/eksctl/pkg/gitops"
	"github.com/weaveworks/eksctl/pkg/gitops/fileprocessor"
)

const (
	defaultGitTimeout = 20 * time.Second
)

// Command creates `generate` commands
func Command(flagGrouping *cmdutils.FlagGrouping) *cobra.Command {
	verbCmd := cmdutils.NewVerbCmd("generate", "Generate GitOps manifests", "")
	cmdutils.AddResourceCmd(flagGrouping, verbCmd, generateProfileCmd)
	return verbCmd
}

type options struct {
	gitops.GitOptions
	ProfilePath       string
	PrivateSSHKeyPath string
}

func generateProfileCmd(rc *cmdutils.Cmd) {
	cfg := api.NewClusterConfig()
	rc.ClusterConfig = cfg

	rc.SetDescription("profile", "Generate a GitOps profile", "")

	var o options

	rc.SetRunFuncWithNameArg(func() error {
		return doGenerateProfile(rc, o)
	})

	rc.FlagSetGroup.InFlagSet("General", func(fs *pflag.FlagSet) {
		fs.StringVarP(&o.URL, "git-url", "", "", "URL for the quickstart base repository")
		fs.StringVarP(&o.Branch, "git-branch", "", "master", "Git branch")
		fs.StringVarP(&o.ProfilePath, "profile-path", "", "./", "Path to generate the profile in")
		_ = cobra.MarkFlagRequired(fs, "git-url")
		fs.StringVar(&o.PrivateSSHKeyPath, "git-private-ssh-key-path", "",
			"Optional path to the private SSH key to use with Git, e.g.: ~/.ssh/id_rsa")

		cmdutils.AddNameFlag(fs, cfg.Metadata)
		cmdutils.AddRegionFlag(fs, rc.ProviderConfig)
		cmdutils.AddConfigFileFlag(fs, &rc.ClusterConfigFile)
	})

	cmdutils.AddCommonFlagsForAWS(rc.FlagSetGroup, rc.ProviderConfig, false)
}

func doGenerateProfile(rc *cmdutils.Cmd, o options) error {
	if err := cmdutils.NewMetadataLoader(rc).Load(); err != nil {
		return err
	}

	processor := &fileprocessor.GoTemplateProcessor{
		Params: fileprocessor.NewTemplateParameters(rc.ClusterConfig),
	}
	profile := &gitops.Profile{
		Processor: processor,
		Path:      o.ProfilePath,
		GitOpts:   o.GitOptions,
		GitCloner: git.NewGitClient(context.Background(), git.ClientParams{
			Timeout:           defaultGitTimeout,
			PrivateSSHKeyPath: o.PrivateSSHKeyPath,
		}),
		FS: afero.NewOsFs(),
		IO: afero.Afero{Fs: afero.NewOsFs()},
	}

	err := profile.Generate(context.Background())

	if err != nil {
		return errors.Wrap(err, "error generating profile")
	}

	return nil
}
