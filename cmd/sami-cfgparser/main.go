package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/asc-ac-at/sam/internal/sami/config"
	"github.com/asc-ac-at/sam/internal/sami/logging/buildlog"
	"github.com/asc-ac-at/sam/internal/sami/sbatch"
)

var (
	cfgFileVar   string
	partitionVar string
	logDirVar    string
)

func init() {
	flag.StringVar(&cfgFileVar, "cfg", "config.yaml", "File containing sbatch config")
	flag.StringVar(&partitionVar, "part", "zen4_gpu", "Slurm partition to target")
	flag.StringVar(&logDirVar, "logdir", "/tmp/sami-logs", "Build log directory used to render the --output path")
}

func main() {
	flag.Parse()

	cfg, err := config.LoadSbatchConfig(cfgFileVar)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// render-only preview: only BuildLog is consulted (for the --output path);
	// nothing is written to disk.
	blPath := &buildlog.BuildLogPaths{BuildLog: logDirVar}

	w := bufio.NewWriter(os.Stdout)
	if err := sbatch.RenderHeaders(&cfg.Sbatch, partitionVar, blPath, w); err != nil {
		log.Fatalln(err)
	}
	w.Flush()
}
