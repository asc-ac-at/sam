package build

import "github.com/spf13/cobra"

var (
	cpuMarch   string
	gpuMarch   string
	gitBranch  string
	gitRepo    string
	swsVariant string
)

// registerFlags registers the set of flags as defined in this file
func registerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&cpuMarch, "cpu", "c", "zen4", "CPU machine architecture, e.g. zen4, zen5")
	cmd.PersistentFlags().StringVarP(&gpuMarch, "gpu", "g", "H100", "GPU machine architecture, e.g. H100, B200")
	cmd.PersistentFlags().StringVarP(&gitBranch, "git-branch", "b", "main", "Remote git branch containing the easybuild command")
	cmd.PersistentFlags().StringVarP(&gitRepo, "git-repo", "r", "https://gitlab.tuwien.ac.at/vsc/software-stacks/asc-software-layer", "Git repository containing easybuild commands")
	cmd.PersistentFlags().StringVarP(&swsVariant, "sws-variant", "s", "2025.06", "Software stack variant or release version")
}
