package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/config"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/sbatch"
)

var (
	cfgFileVar   string
	partitionVar string
)

func init() {
	flag.StringVar(&cfgFileVar, "cfg", "config.yaml", "File containing sbatch config")
	flag.StringVar(&partitionVar, "part", "zen4_gpu", "Slurm partition to target")
}

func main() {
	flag.Parse()

	cfg, err := config.LoadSbatchConfig(cfgFileVar)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	w := bufio.NewWriter(os.Stdout)
	if err := sbatch.RenderHeaders(&cfg.Sbatch, partitionVar, w); err != nil {
		log.Fatalln(err)
	}
	w.Flush()
}
