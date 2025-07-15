package main

import (
	"fmt"
	"io"
	"log"
)

func main() {
	var PackageQuery string
	//var localMaxServerVersions map[string]version.Version
	PackageQuery = "nvidia-graphics-drivers-570"

	//Disabling log

	log.SetOutput(io.Discard)

	nv_utils_max_v, err := getMaxSourceVersionsArchive(PackageQuery)

	if err != nil {
		fmt.Printf("Fatal????")
	}

	PrintSourceVersionMapTable(nv_utils_max_v)

	//log.Printf("%s", localMaxServerVersions["plucky"])

	//////////////////////////////////////////////////////////
	// Get the latest UDA releases from nvidia.com
	// and print them in a table format.
	latestUDAAllEntries, err := GetNvidiaDriverEntries()
	if err != nil {
		log.Fatal(err)
	}
	//PrintTableUDAReleases(latestUDAAllEntries)

	//Grab all of the upstream latest versions from Server. Associate with supportedReleasesJson output.
	//latestServerAllVersions, allBranchesServer, err := getLatestServerDriverVersions()
	_, allBranchesServer, err := getLatestServerDriverVersions()

	if err != nil {
		log.Fatalf("Error fetching driver data: %v", err)
	}
	//printDriverVersions(latestServerAllVersions, allBranchesServer)

	//Evaluate, in order, each one of the nvidia-graphics-drivers-$$branch
	//Compare source files and where is it.

	ubuntuSupportedReleases, err := ReadSupportedReleases("supportedReleases.json")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	UpdateSupportedUDAReleases(latestUDAAllEntries, ubuntuSupportedReleases)
	UpdateSupportedReleasesWithLatestERD(allBranchesServer, ubuntuSupportedReleases)
	// Print or log the updated releases
	PrintSupportedReleases(ubuntuSupportedReleases) // Use PrintSupportedReleases to print to stdout
	////////////////////////////////////////

	for _, release := range ubuntuSupportedReleases {

		currentSourceVersion := "nvidia-graphics-drivers-" + release.BranchName
		currentMaxVersionSource, err := getMaxSourceVersionsArchive(currentSourceVersion)

		if err != nil {
			fmt.Printf("Fatal????")
		}

		PrintSourceVersionMapTableWithSupported(currentMaxVersionSource, ubuntuSupportedReleases)
	}

}
