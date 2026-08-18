package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sbatch SbatchConfig `yaml:"sbatch-config"`
}

// SimpleConfig represents minimal configuration structure
type SbatchConfig struct {
	SharedConfig SharedConfig               `yaml:"shared"`
	Partitions   map[string]PartitionConfig `yaml:"partitions"`
}

// SharedConfig holds any sbatch config that is shared between partitions
// the strings are expected to be a valid sbatch header, example:
//
//	shared:
//	  header: "#sami --vanilla"
//	  footer: |
//	      #SBATCH --job-name="${SW_NAME}-samctr"
//	      #SBATCH --output="$LOGDIR"/slurm-%j.out
type SharedConfig struct {
	Header string `yaml:"header,omitempty"`
	Footer string `yaml:"footer,omitempty"`
}

// ParitionConfig represents the sbatch headers for a specific partition
type PartitionConfig struct {
	Partition      string `yaml:"-"`
	Qos            string `yaml:"qos"`
	Ntasks         string `yaml:"ntasks"`
	Gres           string `yaml:"gres,omitempty"`
	Mem            string `yaml:"mem"`
	CpusPerTask    int    `yaml:"cpus-per-task"`
	ThreadsPerCore int    `yaml:"threads-per-core"`
}

// LoadSbatchConfig is a custom loader that handles mapping partition
// names into the Config.Sbatch struct
func LoadSbatchConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, err
	}
	// single source of truth: name exists only as map key in the file;
	// copy it into the struct here so consumers see a complete value
	for name, p := range c.Sbatch.Partitions {
		p.Partition = name
		c.Sbatch.Partitions[name] = p // write-back required: p is a copy
	}
	return &c, nil
}

var cfgFileVar string

func init() {
	flag.StringVar(&cfgFileVar, "cfg", "config.yaml", "File containing sbatch config")
}

func main() {
	flag.Parse()

	config, err := LoadSbatchConfig(cfgFileVar)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(config.Sbatch.SharedConfig.Header)
	fmt.Println(config.Sbatch.Partitions)
	fmt.Println(config.Sbatch.SharedConfig.Footer)
}
