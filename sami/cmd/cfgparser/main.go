package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sbatch SbatchConfig `yaml:"sbatch-config"`
}

// SimpleConfig represents minimal configuration structure
type SbatchConfig struct {
	SharedConfig    SharedConfig               `yaml:"shared"`
	PartitionConfig map[string]PartitionConfig `yaml:"partitions"`
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
	Partition      string `yaml:"partition"`
	Qos            string `yaml:"qos"`
	Gres           string `yaml:"gres,omitempty"`
	Mem            string `yaml:"mem"`
	CpusPerTask    int    `yaml:"cpus-per-task"`
	ThreadsPerCore int    `yaml:"threads-per-core"`
}

var cfgFileVar string

func init() {
	flag.StringVar(&cfgFileVar, "cfg", "config.yaml", "File containing sbatch config")
}

func main() {
	flag.Parse()

	// Read and parse YAML file
	data, err := os.ReadFile(cfgFileVar)
	if err != nil {
		log.Fatalf("error opening file: %s", cfgFileVar)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var config Config
	if err := dec.Decode(&config); err != nil {
		log.Fatalf("could not unmarshal data: %s", data)
	}
	fmt.Println(config.Sbatch.SharedConfig.Header)
	fmt.Println(config.Sbatch.PartitionConfig)
	fmt.Println(config.Sbatch.SharedConfig.Footer)
}
