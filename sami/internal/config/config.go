package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type File struct {
	Sbatch SbatchConfig `yaml:"sbatch-config"`
}

// SimpleConfig represents minimal configuration structure
type SbatchConfig struct {
	SharedConfig SharedConfig               `yaml:"shared"`
	Partitions   map[string]PartitionConfig `yaml:"partitions"`
}

// SbatchConfig.SbatchHeaders gets the SbatchHeaders for a cpu/gpu configuration
// Returns the headers that target a specific parition or else raises an error
func (s *SbatchConfig) SbatchHeaders(partition string) (*SbatchHeaders, error) {
	p, ok := s.Partitions[partition]
	if !ok {
		return nil, fmt.Errorf("Key not found for partition %s", partition)
	}
	sbheaders := &SbatchHeaders{
		Header:    s.SharedConfig.Header,
		Partition: p,
		Footer:    s.SharedConfig.Footer,
	}
	return sbheaders, nil
}

// SbatchHeaders holds the sbatch configuration for a specific partition
type SbatchHeaders struct {
	Header    string
	Partition PartitionConfig
	Footer    string
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
func LoadSbatchConfig(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var c File
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
