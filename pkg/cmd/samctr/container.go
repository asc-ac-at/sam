// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

/////////////////
// Container
/////////////////

func CtrSetup(rs *RuntimeState, runner func(runtime *RuntimeState) error) (*RuntimeState, error) {
	tmpDir := rs.Storage.RootTmpDir
	url := rs.Image
	sifPath, err := CtrConvertSif(url, tmpDir)
	rs.ContainerSif = sifPath
	if err != nil {
		return rs, err
	}
	// if we have a sif, reuse it
	if _, statErr := os.Stat(sifPath); statErr == nil {
		return rs, nil
	} else if !os.IsNotExist(statErr) {
		return rs, fmt.Errorf("stat %s: %w", sifPath, statErr)
	}

	if err := runner(rs); err != nil {
		return rs, fmt.Errorf("failed to pull container: %w", err)
	}
	// check the file was created
	if _, statErr := os.Stat(sifPath); statErr != nil {
		return rs, fmt.Errorf("container pull completed but %s not found: %w", sifPath, statErr)
	}
	// If a new sif has been created, update the runtime state and return
	return rs, nil
}

func CtrConvertSif(url, tmpDir string) (string, error) {
	if url == "" {
		return url, fmt.Errorf("empty container URL")
	}
	re := regexp.MustCompile(`^(.*://)(.+)$`)
	m := re.FindStringSubmatch(url)
	if len(m) != 3 {
		return "", fmt.Errorf("url format unexpected, matched %d parts in %s", len(url), url)
	}
	imageName := m[2]
	// normalize the sif name
	r := strings.NewReplacer("/", "-", "-", "_", ":", "_")
	sifName := r.Replace(imageName)
	if tmpDir == "" {
		return "", fmt.Errorf("tmpDir undefined")
	}
	full := filepath.Join(tmpDir, sifName+".sif")
	return full, nil
}
