// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	isamctr "github.com/asc-ac-at/sam/internal/samctr"
)

type Config struct {
	ApptainerVarHome     string `mapstructure:"apptainer_var_home"`
	ApptainerVarCachedir string `mapstructure:"apptainer_var_cachedir"`
	//ApptainerMode        string                   `mapstructure:"apptainer_mode"`
	//ApptainerPrg         string                   `mapstructure:"apptainer_prg"`
	//ApptainerPrgArgs     []string                 `mapstructure:"apptainer_prg_args"`
	RootTmpDirPrefix string   `mapstructure:"root_tmp_dir_prefix"`
	BindPaths        []string `mapstructure:"bind_paths"`
	Nvidia           string   `mapstructure:"nvidia"`
	//CtrAdditionalOptions []string                 `mapstructure:"ctr_additional_options"`
	HostInjections string              `mapstructure:"host_injections"`
	Image          string              `mapstructure:"image"`
	FuseMounts     []isamctr.FuseMount `mapstructure:"fusemounts"`
	FuseCmdRW      string              `mapstructure:"fuse_cmd_rw"`
	WriteableRepos []string            `mapstructure:"writeable_repos"`
}

var AppConfig = &Config{}

// initConfig should be registered with cobra.OnInitialize(initConfig)
func initConfig() {
	if err := LoadConfig(cfgFile, RootCmd); err != nil {
		log.Printf("warning: failed to load config: %v", err)
	}
}

func LoadConfig(confPath string, root *cobra.Command) error {
	// defaults
	viper.SetDefault("image", Image)

	// -- config file --
	if confPath != "" {
		viper.SetConfigFile(confPath)
	} else {
		// search for it
		log.Printf("LoadConfig -> searching for config")
		xdg := ""
		xdg = os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" { // fall back to "$HOME/.config"
			// TODO: replace this with an AppName const
			home := os.Getenv("HOME")
			xdg = filepath.Join(home, ".config")
		}
		viper.AddConfigPath(filepath.Join(xdg, "samctr"))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// env var processing
	// TODO: apptainer vars

	// read file if one was found
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config: %w", err)
		}
	} else {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}

	// bind flags to viper
	// Precedence (highest first) is: flag, env var, config file, default.

	if root != nil {
		_ = viper.BindPFlags(root.PersistentFlags())
		_ = viper.BindPFlags(root.Flags())

		// Propagate values into flag.Value

		propagate := func(fs *pflag.FlagSet) {
			fs.VisitAll(func(f *pflag.Flag) {
				if viper.IsSet(f.Name) {
					_ = f.Value.Set(viper.GetString(f.Name))
				}
			})
		}
		propagate(root.PersistentFlags())
		propagate(root.Flags())
	}

	// unmarshall into AppConfig struct
	if err := viper.Unmarshal(AppConfig); err != nil {
		return fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Validate required config fields (fail fast, with useful messages)
	if err := validateRequiredConfig(AppConfig); err != nil {
		return err
	}

	return nil
}

// validateRequiredConfig verifies that required config values are present.
// Add or remove required checks here.
func validateRequiredConfig(c *Config) error {
	// require a default container image
	if strings.TrimSpace(c.Image) == "" {
		return errors.New("configuration error: image is required in config file")
	}

	// require at least one RO fusemount entry (array form)
	if len(c.FuseMounts) == 0 {
		return errors.New("configuration error: fusemounts must be defined in config file and contain at least one entry")
	}

	// ensure entries have required fields
	for i := range c.FuseMounts {
		fmType := strings.TrimSpace(c.FuseMounts[i].Type)
		fmCmd := strings.TrimSpace(c.FuseMounts[i].FuseCmd)
		fmArg := strings.TrimSpace(c.FuseMounts[i].FuseArg)
		fmCtrMt := strings.TrimSpace(c.FuseMounts[i].CtrMountpoint)
		if fmType == "" || fmCmd == "" || fmArg == "" || fmCtrMt == "" {
			return fmt.Errorf("configuration error: fusemounts[%d] must contain type, fuse_cmd and ctr_mountpoint", i)
		}
	}

	// ensure that nvidia mode is implemented
	nvidia_mode := strings.TrimSpace(c.Nvidia)
	if nvidia_mode != "all" {
		// in the future we may use "install,run"
		return fmt.Errorf("configuration error: nvidia mode %s not supported")
	}

	if len(c.WriteableRepos) > 0 {
		// there needs to be a fuse type set, if there is none, use the default
		// (or later fall back on cli flag)
		if strings.TrimSpace(c.FuseCmdRW) == "" {
			FuseCmdRW = "fuse-overlayfs"
		}

	}
	return nil
}
