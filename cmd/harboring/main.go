package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
	"github.com/harboring/harboring/internal/tui"
)

const version = "0.2"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	args := os.Args[1:]
	sub := args[0]

	switch sub {
	case "-h", "--help", "help":
		printUsage()
	case "--version", "-v", "version":
		fmt.Printf("harboring %s\n", version)
	case "view":
		cmdView(args[1:])
	case "dump":
		cmdDump(args[1:])
	case "scrub":
		cmdScrub(args[1:])
	case "shrink":
		cmdShrink(args[1:])
	case "expand":
		cmdExpand(args[1:])
	case "stats":
		cmdStats(args[1:])
	case "perf":
		cmdPerf(args[1:])
	case "validate":
		cmdValidate(args[1:])
	default:
		if strings.HasPrefix(sub, "-") {
			fmt.Fprintf(os.Stderr, "error: unknown flag %s\n", sub)
			os.Exit(1)
		}
		cmdView(args)
	}
}

func printUsage() {
	fmt.Println(`harboring — HTTP Archive Reader In Go

Usage:
  harboring <file.har>              Open TUI viewer
  harboring view <file.har>         Open TUI viewer
  harboring dump <file.har>         Dump formatted JSON to stdout
  harboring scrub <file.har> [-o]   Scrub content sections
  harboring shrink <file.har> [-o]  Minify JSON (remove whitespace)
  harboring expand <file.har> [-o]  Pretty-print JSON
  harboring stats <file.har>        Show summary statistics
  harboring perf <file.har>         Generate performance report
  harboring validate <file.har>     Validate HAR structure
  harboring help                    Show this help

Flags:
  --theme <name|path>  Theme override (TUI mode)
  --scrub              Scrub content sections (view/dump)
  --shrink             Remove whitespace (view/dump)
  --version            Show version
  -o <file>            Output file (scrub/shrink/expand commands)`)
}

func cmdView(args []string) {
	file, themeFlag, doScrub, doShrink := parseViewArgs(args)
	if file == "" {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	root, err := har.ParseFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if doScrub {
		har.ScrubContent(root)
	}
	if doShrink {
		har.ShrinkContent(root)
	}

	th := theme.New()
	th.LoadStartup(themeFlag)

	p := tea.NewProgram(tui.NewApp(root, file, th))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdDump(args []string) {
	file := ""
	doScrub := false
	doShrink := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scrub":
			doScrub = true
		case "--shrink":
			doShrink = true
		default:
			file = args[i]
		}
	}

	if file == "" {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	root, err := har.ParseFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if doScrub {
		har.ScrubContent(root)
	}
	if doShrink {
		har.ShrinkContent(root)
	}

	data, err := har.ExportJSON(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: export: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}

func cmdScrub(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	file := args[0]
	output := ""

	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			output = args[i+1]
			i++
		}
	}

	root, err := har.ParseFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	har.ScrubContent(root)

	data, err := har.ExportJSON(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: export: %v\n", err)
		os.Exit(1)
	}

	if output == "" {
		fmt.Println(string(data))
		return
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write output: %v\n", err)
		os.Exit(1)
	}
}

func cmdShrink(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	file := args[0]
	output := ""

	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			output = args[i+1]
			i++
		}
	}

	if output == "" {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		minified, err := har.MinifyJSON(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(minified))
		return
	}

	if err := har.MinifyFile(file, output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdExpand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	file := args[0]
	output := ""

	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			output = args[i+1]
			i++
		}
	}

	if output == "" {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		expanded, err := har.ExpandJSON(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(expanded))
		return
	}

	if err := har.ExpandFile(file, output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdStats(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	root, err := har.ParseFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	entries := har.GetEntries(root)
	if len(entries) == 0 {
		fmt.Println("No entries found.")
		return
	}

	details := make([]*har.EntryDetails, len(entries))
	for i, e := range entries {
		details[i] = har.GetEntryDetails(e)
	}

	var totalSize int64
	var totalTime float64
	methods := map[string]int{}
	statuses := map[int]int{}

	for _, d := range details {
		totalSize += d.Size
		totalTime += d.Time
		methods[d.Method]++
		statuses[d.Status]++
	}

	fmt.Printf("Requests:    %d\n", len(entries))
	fmt.Printf("Total size:  %s\n", formatBytes(totalSize))
	fmt.Printf("Total time:  %.0f ms\n", totalTime)
	fmt.Println()
	fmt.Println("Methods:")
	for m, c := range methods {
		fmt.Printf("  %-8s %d\n", m, c)
	}
	fmt.Println()
	fmt.Println("Status codes:")
	for s, c := range statuses {
		fmt.Printf("  %d  %d\n", s, c)
	}

	type timedEntry struct {
		detail *har.EntryDetails
		entry  *har.Node
	}
	slowest := make([]timedEntry, 0, len(details))
	for i, d := range details {
		slowest = append(slowest, timedEntry{detail: d, entry: entries[i]})
	}
	for i := 1; i < len(slowest); i++ {
		for j := i; j > 0 && slowest[j].detail.Time > slowest[j-1].detail.Time; j-- {
			slowest[j], slowest[j-1] = slowest[j-1], slowest[j]
		}
	}
	limit := 5
	if len(slowest) < limit {
		limit = len(slowest)
	}
	if limit > 0 {
		fmt.Println()
		fmt.Println("Top 5 slowest:")
		for i := 0; i < limit; i++ {
			d := slowest[i].detail
			fmt.Printf("  %.0f ms  %s %s\n", d.Time, d.Method, d.URL)
		}
	}
}

func cmdPerf(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	root, err := har.ParseFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	entries := har.GetEntries(root)
	if len(entries) == 0 {
		fmt.Println("No entries found.")
		return
	}

	details := make([]*har.EntryDetails, len(entries))
	for i, e := range entries {
		details[i] = har.GetEntryDetails(e)
	}

	var totalSize int64
	var totalTime float64
	var ttfbs []float64
	byType := map[string]int64{}
	byDomain := map[string]int64{}
	byStatus := map[int]int{}
	largest := (*har.EntryDetails)(nil)
	slowestEntry := (*har.EntryDetails)(nil)

	for _, d := range details {
		totalSize += d.Size
		totalTime += d.Time
		if d.Wait > 0 {
			ttfbs = append(ttfbs, d.Wait)
		}
		byType[d.ResourceType] += d.Size
		byDomain[d.Domain] += d.Size
		byStatus[d.Status]++
		if largest == nil || d.Size > largest.Size {
			largest = d
		}
		if slowestEntry == nil || d.Time > slowestEntry.Time {
			slowestEntry = d
		}
	}

	// Compute percentiles
	sort.Float64s(ttfbs)
	p50 := percentile(ttfbs, 50)
	p90 := percentile(ttfbs, 90)
	p95 := percentile(ttfbs, 95)
	p99 := percentile(ttfbs, 99)

	fmt.Printf("HAR Performance Report\n")
	fmt.Printf("========================\n\n")
	fmt.Printf("Requests:        %d\n", len(entries))
	fmt.Printf("Total size:      %s\n", formatBytes(totalSize))
	fmt.Printf("Total load time: %.0f ms\n", totalTime)
	fmt.Println()

	fmt.Println("TTFB Percentiles:")
	fmt.Printf("  p50: %.0f ms\n", p50)
	fmt.Printf("  p90: %.0f ms\n", p90)
	fmt.Printf("  p95: %.0f ms\n", p95)
	fmt.Printf("  p99: %.0f ms\n", p99)
	fmt.Println()

	fmt.Println("By Resource Type:")
	type typeKV struct {
		k string
		v int64
	}
	types := make([]typeKV, 0, len(byType))
	for t, s := range byType {
		if t == "" {
			t = "other"
		}
		types = append(types, typeKV{k: t, v: s})
	}
	sort.Slice(types, func(i, j int) bool { return types[i].v > types[j].v })
	for _, t := range types {
		fmt.Printf("  %-12s %s\n", t.k+":", formatBytes(t.v))
	}
	fmt.Println()

	fmt.Println("By Domain (top 10):")
	type kv struct {
		k string
		v int64
	}
	domains := make([]kv, 0, len(byDomain))
	for k, v := range byDomain {
		domains = append(domains, kv{k, v})
	}
	sort.Slice(domains, func(i, j int) bool { return domains[i].v > domains[j].v })
	for i, d := range domains {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-30s %s\n", d.k+":", formatBytes(d.v))
	}
	fmt.Println()

	fmt.Println("Status Codes:")
	for s, c := range byStatus {
		fmt.Printf("  %d: %d\n", s, c)
	}
	fmt.Println()

	fmt.Println("Largest resource:")
	if largest != nil {
		fmt.Printf("  %s %s (%s)\n", largest.Method, largest.URL, formatBytes(largest.Size))
	}
	fmt.Println("Slowest resource:")
	if slowestEntry != nil {
		fmt.Printf("  %.0f ms  %s %s\n", slowestEntry.Time, slowestEntry.Method, slowestEntry.URL)
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func cmdValidate(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	root, err := har.ParseFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if root.Kind != har.Object {
		fmt.Println("INVALID: root is not a JSON object")
		os.Exit(1)
	}

	logNode := findChild(root, "log")
	if logNode == nil {
		fmt.Println("INVALID: missing 'log' key at root")
		os.Exit(1)
	}

	entriesNode := findChild(logNode, "entries")
	if entriesNode == nil {
		fmt.Println("INVALID: missing 'entries' in log")
		os.Exit(1)
	}
	if entriesNode.Kind != har.Array {
		fmt.Println("INVALID: 'entries' is not an array")
		os.Exit(1)
	}

	for i, entry := range entriesNode.Children {
		if findChild(entry, "request") == nil {
			fmt.Printf("INVALID: entry [%d] missing 'request'\n", i)
			os.Exit(1)
		}
		if findChild(entry, "response") == nil {
			fmt.Printf("INVALID: entry [%d] missing 'response'\n", i)
			os.Exit(1)
		}
	}

	fmt.Printf("VALID: %d entries, all have request and response\n", len(entriesNode.Children))
}

func findChild(n *har.Node, key string) *har.Node {
	for _, child := range n.Children {
		if child.Key == key {
			return child
		}
	}
	return nil
}

func parseViewArgs(args []string) (string, string, bool, bool) {
	file := ""
	themeFlag := ""
	doScrub := false
	doShrink := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--theme":
			if i+1 < len(args) {
				themeFlag = args[i+1]
				i++
			}
		case "--scrub":
			doScrub = true
		case "--shrink":
			doShrink = true
		default:
			file = args[i]
		}
	}
	return file, themeFlag, doScrub, doShrink
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
