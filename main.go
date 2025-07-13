package main

import (
	"fmt"
	"io"
	"log"
)

func main() {
	//var PackageQuery string
	//var localMaxServerVersions map[string]version.Version
	//PackageQuery = "libnvidia-encode-570"

	//Disabling log

	log.SetOutput(io.Discard)

	// nv_utils_max_v, err := getMaxVersionsArchive(PackageQuery)

	// if err != nil {
	// 	fmt.Printf("Fatal????")
	// }

	// PrintVersionMapTable(nv_utils_max_v)

	//log.Printf("%s", localMaxServerVersions["plucky"])

	entries, err := GetNvidiaDriverEntries()
	if err != nil {
		log.Fatal(err)
	}

	PrintAlludaReleases(entries)
	//Grab all of the upstream latest versions from Server. Associate with supportedReleasesJson output.

	latest, allBranches, err := getLatestServerDriverVersions()
	if err != nil {
		log.Fatalf("Error fetching driver data: %v", err)
	}

	//logDriverVersions(latest, allBranches)
	// or
	printDriverVersionsToStdout(latest, allBranches)

	//Grab all of the upstream latest versions from Unix. Associate with supportedReleasesJson output.
	//Get the latest publication date. Associate with supportedReleasesJson output.
	//Evaluate, in order, each one of the nvidia-graphics-drivers-$$branch
	//Compare source files and where is it.

	releases, err := ReadSupportedReleases("supportedReleases.json")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	PrintSupportedReleases(releases)
	UpdateSupportedUDAReleases(entries, releases)
	PrintSupportedReleases(releases)
	UpdateSupportedERDReleases(releases, branchInfo)

	// Print or log the updated releases
	PrintSupportedReleases(releases) // Use PrintSupportedReleases to print to stdout

	// Or use LogSupportedReleases(releases) if you want to log instead of print
}
