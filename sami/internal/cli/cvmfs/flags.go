package cvmfs

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	ctrTool string
	publish bool
)

func registerFlags(cmd *cobra.Command) (*CvmfsBuildCmdData, error) {
	cmdData := NewCvmfsBuildCmdData()

	cpuMarch, _ := cmd.InheritedFlags().GetString("cpu")
	gpuMarch, _ := cmd.InheritedFlags().GetString("gpu")
	gitBranch, _ := cmd.InheritedFlags().GetString("git-branch")
	gitRepo, _ := cmd.InheritedFlags().GetString("git-repo")
	swsVariant, _ := cmd.InheritedFlags().GetString("sws-variant")
	publish, _ := cmd.Flags().GetBool("publish")

	cmdData.CpuMarch = cpuMarch
	cmdData.GpuMarch = gpuMarch
	cmdData.GitBranch = gitBranch
	cmdData.GitRepo = gitRepo
	cmdData.SwsVariant = swsVariant

	cmdData.Easystack = fmt.Sprintf("./easystacks/%s/asc_eb_5.3.0-2024a.yaml", cmdData.SwsVariant)
	cmdData.LmodInit = fmt.Sprintf("/cvmfs/software.eessi.io/versions/%s/init/lmod/sh", cmdData.SwsVariant)

	cmdData.Publish = publish
	return cmdData, nil
}
