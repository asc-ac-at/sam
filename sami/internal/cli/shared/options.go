package shared

import "github.com/spf13/cobra"

// Options is a mutable object containing data collected from use specified command line arguments
type Options struct {
	CPU              string `flag:"cpu" default:"zen4"`
	GPU              string `flag:"gpu" default:"h100"`
	GitBranch        string
	GitCommit        string
	GitRepo          string   `flag:"git-repo" default:"https://gitlab.tuwien.ac.at/vsc/software-stacks/asc-software-layer"`
	GitMergeReqId    int      `flag:"git-mr-id" default:"0"`
	SWSVariant       string   `flag:"sws-variant" default:"2025.06"`
	Name             string   `flag:"name"`
	BuildLogBasePath string   `flag:"log-basepath" default:"/opt/adm/asc-software-stack"`
	Verbose          bool     `flag:"verbose" default:"false"`
	Files            []string `flag:"files"`
}

func NewOptions() *Options {
	return &Options{
		CPU:              "zen4",
		GPU:              "h100",
		GitRepo:          "https://gitlab.tuwien.ac.at/vsc/software-stacks/asc-software-layer",
		SWSVariant:       "2025.06",
		BuildLogBasePath: "/opt/adm/asc-software-stack",
	}
}

func RegisterFlags(cmd *cobra.Command, opts *Options) *Options {
	cmd.PersistentFlags().StringVarP(&opts.CPU, "cpu", "c", opts.CPU, "CPU machine architecture")
	cmd.PersistentFlags().StringVarP(&opts.GPU, "gpu", "g", opts.GPU, "GPU machine architecture")
	cmd.PersistentFlags().StringVar(&opts.GitBranch, "git-branch", opts.GitBranch, "Remote git branch")
	cmd.PersistentFlags().StringVar(&opts.GitCommit, "git-commit", opts.GitCommit, "Hash of the git commit to retrieve")
	cmd.PersistentFlags().StringVar(&opts.GitRepo, "git-repo", opts.GitRepo, "Git repository URL")
	cmd.PersistentFlags().IntVar(&opts.GitMergeReqId, "git-mr-id", opts.GitMergeReqId, "GitLab Merge Request ID")
	cmd.PersistentFlags().StringVarP(&opts.SWSVariant, "sws-variant", "s", opts.SWSVariant, "Software stack variant")
	cmd.PersistentFlags().StringVarP(&opts.Name, "name", "n", opts.Name, "Name for the software build")
	cmd.PersistentFlags().StringVar(&opts.BuildLogBasePath, "log-basepath", opts.BuildLogBasePath, "Sets the base of the build log directory tree")
	cmd.PersistentFlags().BoolVar(&opts.Verbose, "verbose", opts.Verbose, "Enable verbose logging")
	cmd.PersistentFlags().StringArrayVarP(&opts.Files, "files", "f", opts.Files, "Easystack files to pass to easybuild. Overrides any changed files in repo.")
	return opts
}
