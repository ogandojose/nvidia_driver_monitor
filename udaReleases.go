package main

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"fmt"

	"golang.org/x/net/html"
)

type DriverEntry struct {
	Version string
	Date    time.Time
	IsBeta  bool
}

// Print to standard output using fmt
func (d DriverEntry) Print() {
	fmt.Printf("Version: %s | Date: %s | Beta: %t\n", d.Version, d.Date.Format("2006-01-02"), d.IsBeta)
}

// Log to stdout using log.Println
func (d DriverEntry) Log() {
	log.Printf("Version: %s | Date: %s | Beta: %t", d.Version, d.Date.Format("2006-01-02"), d.IsBeta)
}

// PrintAll prints all DriverEntries to standard output using fmt.Println
func PrintAlludaReleases(entries []DriverEntry) {
	for _, entry := range entries {
		entry.Print()
	}
}

// LogAll logs all DriverEntries using log.Println
func LogAlludaReleases(entries []DriverEntry) {
	for _, entry := range entries {
		entry.Log()
	}
}

func GetNvidiaDriverEntries() ([]DriverEntry, error) {
	url := "https://www.nvidia.com/en-us/drivers/unix/linux-amd64-display-archive/"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	pressRoom := findPressRoom(root)
	if pressRoom == nil {
		return nil, fmt.Errorf("pressRoom div not found")
	}

	return extractDriverEntries(pressRoom), nil
}

func findPressRoom(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "div" {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == "pressRoom" {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if res := findPressRoom(c); res != nil {
			return res
		}
	}
	return nil
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func parseDriverEntryDiv(div *html.Node) *DriverEntry {
	var version string
	var date time.Time
	isBeta := false

	var reVersion = regexp.MustCompile(`Version:\s*(\d{3}\.\d{1,3}(?:\.\d{1,3})?)`)
	var reDate = regexp.MustCompile(`Release Date:\s*([A-Za-z]+ \d{1,2}, \d{4})`)

	var buf strings.Builder

	var gatherText func(*html.Node)
	gatherText = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			gatherText(c)
		}
	}

	// Helper to check for BETA <sup> or text
	var containsBeta func(*html.Node) bool
	containsBeta = func(n *html.Node) bool {
		if n.Type == html.TextNode && strings.Contains(strings.ToUpper(n.Data), "BETA") {
			return true
		}
		if n.Type == html.ElementNode && n.Data == "sup" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if containsBeta(c) {
					return true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if containsBeta(c) {
				return true
			}
		}
		return false
	}

	for c := div.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "p" {
			buf.Reset()
			gatherText(c)

			text := buf.String()

			matchVer := reVersion.FindStringSubmatch(text)
			matchDate := reDate.FindStringSubmatch(text)

			// Check for BETA in the <p> node
			if containsBeta(c) {
				isBeta = true
			}

			if len(matchVer) >= 2 && len(matchDate) >= 2 {
				version = matchVer[1]
				parsedDate, err := time.Parse("January 2, 2006", matchDate[1])
				if err == nil {
					date = parsedDate
					return &DriverEntry{Version: version, Date: date, IsBeta: isBeta}
				}
			}
		}
	}

	return nil
}

func extractDriverEntries(n *html.Node) []DriverEntry {
	var entries []DriverEntry

	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" {
			class := getAttr(node, "class")
			if class == "pressItem" || class == "driver-info" {
				entry := parseDriverEntryDiv(node)
				if entry != nil {
					entries = append(entries, *entry)
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(n)
	return entries
}

// UpdateSupportedReleases updates the CurrentUpstreamVersion field of each SupportedRelease
// with the latest DriverEntry version matching the major version from BranchName.
func UpdateSupportedUDAReleases(entries []DriverEntry, releases []SupportedRelease) []SupportedRelease {
	// Build a map: major version -> latest non-beta DriverEntry
	latestByMajor := make(map[string]DriverEntry)
	for _, entry := range entries {
		if entry.IsBeta {
			continue
		}
		major := strings.SplitN(entry.Version, ".", 2)[0]
		if prev, ok := latestByMajor[major]; !ok || entry.Date.After(prev.Date) {
			latestByMajor[major] = entry
		}
	}

	// Update releases
	for i, rel := range releases {
		major := rel.BranchName
		if entry, ok := latestByMajor[major]; ok {
			releases[i].CurrentUpstreamVersion = entry.Version
		}
	}
	return releases
}
