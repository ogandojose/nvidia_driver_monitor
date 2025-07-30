package drivers

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

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
func GetNvidiaDriverEntries() ([]DriverEntry, error) {
	url := "https://www.nvidia.com/en-us/drivers/unix/linux-amd64-display-archive/"

	resp, err := utils.HTTPGetWithRetry(url)
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

// Helper functions for HTML parsing
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
