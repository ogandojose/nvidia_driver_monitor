package drivers

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/utils"

	"golang.org/x/net/html"
)

// DriverEntry represents a driver entry from NVIDIA's website
type DriverEntry struct {
	Version string
	Date    time.Time
	IsBeta  bool
}

// PrintTableUDAReleases prints all DriverEntries in a table format to standard output
func PrintTableUDAReleases(entries []DriverEntry) {
	fmt.Println("These are the latest nvidia.com UDA releases:")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Version\tDate\tBeta")
	for _, entry := range entries {
		fmt.Fprintf(w, "%s\t%s\t%t\n", entry.Version, entry.Date.Format("2006-01-02"), entry.IsBeta)
	}
	w.Flush()
	fmt.Println("----------------------------------------------------")
}

// LogTableUDAReleases logs all DriverEntries in a table format using log.Println
func LogTableUDAReleases(entries []DriverEntry) {
	log.Println("These are the latest nvidia.com UDA releases:")

	var b strings.Builder
	b.WriteString("Version\tDate\tBeta\n")
	for _, entry := range entries {
		fmt.Fprintf(&b, "%s\t%s\t%t\n", entry.Version, entry.Date.Format("2006-01-02"), entry.IsBeta)
	}
	log.Print("\n" + b.String())
	log.Println("----------------------------------------------------")
}

// GetNvidiaDriverEntries retrieves driver entries from NVIDIA's website
// branchMajors limits directory traversal to the supplied major versions (e.g. "580")
func GetNvidiaDriverEntries(cfg *config.Config, branchMajors []string) ([]DriverEntry, error) {
	baseURL := ensureTrailingSlash(cfg.URLs.NVIDIA.DriverArchiveURL)

	resp, err := utils.HTTPGetWithRetry(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch driver directory index: %w", err)
	}
	defer resp.Body.Close()

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse driver directory HTML: %w", err)
	}

	versionDirs := extractDriverDirectories(root)
	if len(versionDirs) == 0 {
		return nil, fmt.Errorf("no driver directories found at %s", baseURL)
	}

	selectedDirs := selectDirectoriesByBranches(versionDirs, branchMajors)
	if len(selectedDirs) == 0 {
		selectedDirs = versionDirs
	}

	entries := make([]DriverEntry, 0, len(selectedDirs))
	for _, dir := range selectedDirs {
		entry, err := buildDriverEntry(baseURL, dir)
		if err != nil {
			log.Printf("failed to build UDA entry for %s: %v", dir, err)
			continue
		}
		entries = append(entries, *entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Date.Equal(entries[j].Date) {
			return entries[i].Version > entries[j].Version
		}
		return entries[i].Date.After(entries[j].Date)
	})

	return entries, nil
}

func ensureTrailingSlash(url string) string {
	if strings.HasSuffix(url, "/") {
		return url
	}
	return url + "/"
}

func extractDriverDirectories(root *html.Node) []string {
	var dirs []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" && getAttr(n, "class") == "dir" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "a" {
					href := getAttr(c, "href")
					if strings.HasSuffix(href, "/") && href != "../" {
						dirs = append(dirs, href)
					}
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)
	return dirs
}

func buildDriverEntry(baseURL, directory string) (*DriverEntry, error) {
	dirURL := baseURL + directory

	resp, err := utils.HTTPGetWithRetry(dirURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch directory %s: %w", dirURL, err)
	}
	defer resp.Body.Close()

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory HTML for %s: %w", dirURL, err)
	}

	licenseDate, err := findLicenseDate(root)
	if err != nil {
		return nil, fmt.Errorf("failed to extract license.txt timestamp from %s: %w", dirURL, err)
	}

	version := strings.TrimSuffix(directory, "/")
	isBeta := strings.Contains(strings.ToLower(version), "beta")

	return &DriverEntry{Version: version, Date: licenseDate, IsBeta: isBeta}, nil
}

func findLicenseDate(root *html.Node) (time.Time, error) {
	var (
		parsedTime time.Time
		found      bool
		parseErr   error
	)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found {
			return
		}

		if n.Type == html.ElementNode && n.Data == "span" && getAttr(n, "class") == "file" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "a" && getAttr(c, "href") == "license.txt" {
					dateNode := findSiblingDate(n)
					if dateNode == nil {
						parseErr = fmt.Errorf("license.txt date not found")
						found = true
						return
					}

					timestamp := strings.TrimSpace(collectText(dateNode))
					if timestamp == "" {
						parseErr = fmt.Errorf("license.txt timestamp empty")
						found = true
						return
					}

					var err error
					parsedTime, err = time.Parse("2006-01-02 15:04", timestamp)
					if err != nil {
						parseErr = fmt.Errorf("invalid license.txt timestamp %q: %w", timestamp, err)
					}
					found = true
					return
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
			if found {
				return
			}
		}
	}

	walk(root)

	if !found {
		return time.Time{}, fmt.Errorf("license.txt date not found")
	}

	if parseErr != nil {
		return time.Time{}, parseErr
	}

	return parsedTime, nil
}

func findSiblingDate(node *html.Node) *html.Node {
	for sibling := node.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == html.TextNode {
			continue
		}
		if sibling.Type == html.ElementNode && sibling.Data == "span" && getAttr(sibling, "class") == "date" {
			return sibling
		}
	}

	if parent := node.Parent; parent != nil {
		for sibling := parent.FirstChild; sibling != nil; sibling = sibling.NextSibling {
			if sibling.Type == html.ElementNode && sibling.Data == "span" && getAttr(sibling, "class") == "date" {
				return sibling
			}
		}
	}

	return nil
}

func collectText(n *html.Node) string {
	var b strings.Builder

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(n)
	return b.String()
}

type directorySelection struct {
	dir   string
	parts []int
	beta  bool
}

func selectDirectoriesByBranches(versionDirs []string, branchMajors []string) []string {
	if len(branchMajors) == 0 {
		return nil
	}

	targets := make(map[string]struct{}, len(branchMajors))
	for _, major := range branchMajors {
		major = strings.TrimSpace(major)
		if major == "" {
			continue
		}
		targets[major] = struct{}{}
	}

	if len(targets) == 0 {
		return nil
	}

	best := make(map[string]directorySelection)

	for _, dir := range versionDirs {
		version := normalizeDirectoryVersion(dir)
		if version == "" {
			continue
		}
		major := extractMajorVersion(version)
		if major == "" {
			continue
		}

		if _, ok := targets[major]; !ok {
			continue
		}

		parts := parseVersionParts(version)
		if len(parts) == 0 {
			continue
		}

		beta := isBetaDirectory(dir)

		current, exists := best[major]
		if !exists {
			best[major] = directorySelection{dir: dir, parts: parts, beta: beta}
			continue
		}

		// Prefer GA over beta when versions are comparable
		if current.beta && !beta {
			best[major] = directorySelection{dir: dir, parts: parts, beta: beta}
			continue
		}
		if current.beta == beta && compareVersionParts(parts, current.parts) > 0 {
			best[major] = directorySelection{dir: dir, parts: parts, beta: beta}
			continue
		}
		if !current.beta && beta {
			// Keep GA unless beta is strictly newer than existing GA version
			if compareVersionParts(parts, current.parts) > 0 {
				best[major] = directorySelection{dir: dir, parts: parts, beta: beta}
			}
		}
	}

	if len(best) == 0 {
		return nil
	}

	results := make([]string, 0, len(best))
	for _, major := range branchMajors {
		if sel, ok := best[strings.TrimSpace(major)]; ok {
			results = append(results, sel.dir)
		}
	}

	return results
}

func normalizeDirectoryVersion(directory string) string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(directory, "/"))
	if trimmed == "" {
		return ""
	}

	parts := strings.Split(trimmed, "/")
	candidate := parts[len(parts)-1]

	segments := strings.Split(candidate, "-")
	candidate = segments[len(segments)-1]
	candidate = strings.TrimSpace(candidate)

	return candidate
}

func extractMajorVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}

	if idx := strings.IndexRune(version, '.'); idx >= 0 {
		major := version[:idx]
		return strings.TrimLeft(major, "0")
	}

	// Fallback: accumulate leading digits
	var sb strings.Builder
	for _, r := range version {
		if r < '0' || r > '9' {
			break
		}
		sb.WriteRune(r)
	}

	return strings.TrimLeft(sb.String(), "0")
}

func parseVersionParts(version string) []int {
	var cleaned strings.Builder
	for _, r := range version {
		if (r >= '0' && r <= '9') || r == '.' {
			cleaned.WriteRune(r)
			continue
		}
		if cleaned.Len() > 0 {
			break
		}
	}

	value := cleaned.String()
	if value == "" {
		return nil
	}

	pieces := strings.Split(value, ".")
	parts := make([]int, 0, len(pieces))
	for _, piece := range pieces {
		if piece == "" {
			parts = append(parts, 0)
			continue
		}
		num, err := strconv.Atoi(piece)
		if err != nil {
			return nil
		}
		parts = append(parts, num)
	}

	return parts
}

func compareVersionParts(a, b []int) int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	for i := 0; i < maxLen; i++ {
		var ai, bi int
		if i < len(a) {
			ai = a[i]
		}
		if i < len(b) {
			bi = b[i]
		}

		if ai > bi {
			return 1
		}
		if ai < bi {
			return -1
		}
	}

	return 0
}

func isBetaDirectory(dir string) bool {
	dir = strings.ToLower(dir)
	return strings.Contains(dir, "beta")
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
