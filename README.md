# SAM - Software and Modules

This repo contains some programs used to generate the containerized
build environments used by the software and modules working group at the
ASC Research Center. The sam working group uses EESSI[^1] as a base for
the software stack that is provided on the MUSICA cluster.

# crtar

This is a simple program that can be used to create tarballs, it is based on a
bash function defined in the `build_container.sh` script used on the harbok
cluster at University of Groeningen[^2].

# samctr

A simple wrapper around some apptainer commands. Why the wrapper? The
use case that is being targeted by `samctr` is to create a portable
build environment for cvmfs[^3] repositories, ahead of publishing newly
built software to a stratum0. 

Part of the motivation for creating this program was to make it easier
to submit software build + publish jobs as `sbatch` scripts.

Most of the login implemented in `samctr` is derived from the
`eessi_container.sh` script authored by Thomas Roeblitz[^4].

## Examples

### config.yaml

`samctr` will look for a config file (in order of precedence) at: 
+ the path specified after the `--config` or `-f` flags
+ `$XDG_CONFIG_HOME/samctr/config.yaml`
+ `$HOME/.config/samctr/config.yaml`

A config file can be used to configure various options that will be
passed through to the `apptainer` commands later run by the program.
For example, the following will pull a specific image create a sif
file therof, set up a bind mount for `host_injections` and (read only)
fusemounts for a number of cvmfs. Additionally, a writeable overlay for
one of these repositories is set up. Also any nvidia gpu found on the
host will be passed through to the container.

```
image: "docker://registry.somewhere/builder-rocky96:24601"
host_injections: /opt/alt/eessi
fusemounts:
  - type: "container"
    fuse_cmd: "cvmfs2"
    fuse_arg: "cvmfs-config.cern.ch"
    ctr_mountpoint: "/cvmfs/cvmfs-config.cern.ch"
  - type: "container"
    fuse_cmd: "cvmfs2"
    fuse_arg: "software.eessi.io"
    ctr_mountpoint: "/cvmfs/software.eessi.io"
  - type: "container"
    fuse_cmd: "cvmfs2"
    fuse_arg: "speedy.repo"
    ctr_mountpoint: "/cvmfs_ro/speedy.repo"
nvidia: all
writeable_repos: ["speedy.repo"]
fuse_cmd_rw: fuse-overlayfs
```

### shell

This can be used to launch an interactive shell with a specific config.

```
$ samctr shell --config=/tmp/min-conf.yaml
```

### exec

This wraps the "apptainer exec" command, the `--` seperator is used
to signify what should be passed on to `apptainer exec` as program
arguments.


#### simple command 

```
$ samctr exec -- ls /cvmfs
```
	
#### interpret an arbitrary script

```
$ cat hello_world.py | samctr exec -- python3
```

#### Build piece of software

```
$ cat >build_go_1250.sh <<EOF
#!/bin/sh

export EESSI_PROJECT_INSTALL=/cvmfs/software.asc.ac.at
source /cvmfs/software.eessi.io/versions/2023.06/init/lmod/bash
source /cvmfs/software.eessi.io/versions/2023.06/init/bash
module load EESSI-extend
eb -r Go-1.25.0.eb
EOF
$ samctr exec -- /bin/sh <build_go_1250.sh
```

#### Submit a build as a slurm job

Define a build script something along the following lines

```
> cat build_go1250_jobscript.sh
#!/bin/sh
#SBATCH -p zen4_0768
#SBATCH --mem=64G
#SBATCH --ntasks=1

# Define a build command to pass into the container
cat >tmp_build_cmd.sh << EOBC
#!/bin/sh
export EESSI_PROJECT_INSTALL=/cvmfs/software.asc.ac.at
source /cvmfs/software.eessi.io/versions/2023.06/init/lmod/bash
source /cvmfs/software.eessi.io/versions/2023.06/init/bash

module load EESSI-extend
eb -r Go-1.25.0.eb

# Create tarabll shared directory to access after job completion
crtar -name Go-1.25.0 -outputDir /opt/adm/sw-archives
EOBC
chmod +x tmp_build_cmd.sh

# run the "build_cmd" in apptainer with default config
samctr exec -- /bin/sh <tmp_build_cmd.sh

# cleanup
rm -f tmp_build_cmd.sh
```

submit the job with sbatch:
```
sbatch build_go1250_jobscript.sh
```


# Footnotes

[^1]: European Environment For Scientific Software Installations <https://eessi.io>
[^2]: <https://gitrepo.service.rug.nl/cit-hpc/habrok/cit-hpc-easybuild/-/blob/main/jobscripts/habrok/build_container.sh?ref_type=heads>
[^3]: Cern VM-FS <https://cernvm.cern.ch/fs/>
[^4]: <https://github.com/EESSI/software-layer-scripts/blob/main/eessi_container.sh>
