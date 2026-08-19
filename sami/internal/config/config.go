package config

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// GenericArchSubdir is the EESSI subdir for generic (non-tuned) x86_64
// builds, selected with the --generic flag instead of --arch.
const GenericArchSubdir = "x86_64/generic"

type File struct {
	Sbatch       SbatchConfig      `yaml:"sbatch-config"`
	ArchMapping  map[string]string `yaml:"arch-mapping"`
	AccelMapping map[string]string `yaml:"accel-mapping"`
}

// SimpleConfig represents minimal configuration structure
type SbatchConfig struct {
	SharedConfig SharedConfig               `yaml:"shared"`
	Partitions   map[string]PartitionConfig `yaml:"partitions"`
}

// ArchSubdir resolves a short arch name (e.g. zen4) to its EESSI
// subdirectory (e.g. x86_64/amd/zen4) using the arch-mapping table.
func (f *File) ArchSubdir(arch string) (string, error) {
	sub, ok := f.ArchMapping[arch]
	if !ok {
		return "", fmt.Errorf("unknown arch %q; valid archs: %s", arch, strings.Join(sortedKeys(f.ArchMapping), ", "))
	}
	return sub, nil
}

// AccelSubdir resolves a short accelerator name (e.g. cc90) to its EESSI
// subdirectory (e.g. accel/nvidia/cc90) using the accel-mapping table.
func (f *File) AccelSubdir(accel string) (string, error) {
	sub, ok := f.AccelMapping[accel]
	if !ok {
		return "", fmt.Errorf("unknown accel %q; valid accels: %s", accel, strings.Join(sortedKeys(f.AccelMapping), ", "))
	}
	return sub, nil
}

// SbatchConfig.SbatchHeaders gets the SbatchHeaders for a cpu/gpu configuration
// Returns the headers that target a specific parition or else raises an error
func (s *SbatchConfig) SbatchHeaders(partition string) (*SbatchHeaders, error) {
	p, ok := s.Partitions[partition]
	if !ok {
		return nil, fmt.Errorf("unknown partition %q; valid partitions: %s", partition, strings.Join(s.validPartitionNames(), ", "))
	}
	sbheaders := &SbatchHeaders{
		Header:    s.SharedConfig.Header,
		Partition: p,
		Footer:    s.SharedConfig.Footer,
	}
	return sbheaders, nil
}

// validPartitionNames returns the configured partition names in sorted order,
// so error messages are deterministic regardless of map iteration order.
func (s *SbatchConfig) validPartitionNames() []string {
	return sortedKeys(s.Partitions)
}

// sortedKeys returns the keys of a map in sorted order, for deterministic
// error messages regardless of map iteration order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
