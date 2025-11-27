// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package crtar

// Real test case for FindModules 
// {EESSI 2023.06} Apptainer> find  /tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4/modules/ -type f
// /tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4/modules/all/Go/1.25.0.lua


// Test case for FindSoftware
// {EESSI 2023.06} Apptainer> export ARCHDIR=/tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4
// {EESSI 2023.06} Apptainer> find ${ARCHDIR}/software/*/* -maxdepth 1 -name easybuild -type d
// /tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4/software/Go/1.25.0/easybuild
// {EESSI 2023.06} Apptainer> find ${ARCHDIR}/software/*/* -maxdepth 1 -name easybuild -type d | xargs -r dirname
// /tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4/software/Go/1.25.0
