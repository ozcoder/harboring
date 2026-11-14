package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/report"
	"github.com/harboring/harboring/internal/theme"
	"github.com/harboring/harboring/internal/tui"
)

const version = "0.4.0"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if opts.help {
		printUsage()
		return
	}
	if opts.version {
		fmt.Printf("harboring %s\n", version)
		return
	}

	// --list is an informational, standalone command.
	if opts.list != "" {
		switch opts.list {
		case "themes":
			listThemes()
		case "trackers":
			listTrackers()
		}
		return
	}

	switch opts.mode {
	case "", "view":
		cmdView(opts)
	case "dump":
		cmdDump(opts)
	case "scrub":
		cmdScrub(opts)
	case "shrink":
		cmdShrink(opts)
	case "expand":
		cmdExpand(opts)
	case "stats":
		cmdStats(opts)
	case "perf":
		cmdPerf(opts)
	case "cache":
		cmdCache(opts)
	case "security":
		cmdSecurity(opts)
	case "trackers":
		cmdTrackers(opts)
	case "priority":
		cmdPriority(opts)
	case "validate":
		cmdValidate(opts)
	case "audit":
		cmdAudit(opts)
	case "fields":
		cmdFields(opts)
	case "diff":
		cmdDiff(opts)
	}
}

// options holds the parsed command line. mode selects the operation and is one
// of "", "view", "dump", "scrub", "shrink", "expand", "stats", "perf", "cache",
// "security", "trackers", "priority", "validate", "fields", or "diff"; an empty mode means the default viewer.
type options struct {
	mode         string
	file         string
	file2        string
	theme        string
	output       string
	markdown     bool
	jsonOut      bool
	help         bool
	version      bool
	fields       string
	list         string
	trackersFile string
	audit        string
	auditLargeKb int64
}

// parseArgs converts command line flags into options. Operations are selected
// with a single --<name> flag; bare positional arguments are input files.
func parseArgs(args []string) (*options, error) {
	o := &options{}
	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			o.help = true
		case "-v", "--version":
			o.version = true
		case "-m", "--markdown":
			o.markdown = true
		case "-j", "--json":
			o.jsonOut = true
		case "--view", "--dump", "--scrub", "--shrink", "--expand",
			"--stats", "--perf", "--cache", "--security", "--trackers", "--priority", "--validate", "--audit":
			if a == "--audit" {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("--audit requires an argument (e.g. images)")
				}
				mode := "audit"
				if o.mode != "" && o.mode != mode {
					return nil, fmt.Errorf("conflicting modes: --%s and --%s", o.mode, mode)
				}
				o.mode = mode
				o.audit = args[i+1]
				if o.audit != "images" && o.audit != "redirects" {
					return nil, fmt.Errorf("--audit: unknown type %q (expected \"images\" or \"redirects\")", o.audit)
				}
				i++
				continue
			}
			mode := a[2:]
			if o.mode != "" && o.mode != mode {
				return nil, fmt.Errorf("conflicting modes: --%s and --%s", o.mode, mode)
			}
			o.mode = mode
		case "--diff":
			if o.mode != "" && o.mode != "diff" {
				return nil, fmt.Errorf("conflicting modes: --%s and --diff", o.mode)
			}
			o.mode = "diff"
		case "--fields":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--fields requires a comma-separated list of fields")
			}
			o.fields = args[i+1]
			if o.mode != "" && o.mode != "fields" {
				return nil, fmt.Errorf("conflicting modes: --%s and --fields", o.mode)
			}
			o.mode = "fields"
			i++
		case "--theme":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--theme requires an argument")
			}
			o.theme = args[i+1]
			i++
		case "--list":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--list requires an argument (themes or trackers)")
			}
			o.list = args[i+1]
			if o.list != "themes" && o.list != "trackers" {
				return nil, fmt.Errorf("--list: unknown category %q (expected \"themes\" or \"trackers\")", o.list)
			}
			i++
		case "--trackers-file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--trackers-file requires a file path")
			}
			o.trackersFile = args[i+1]
			i++
		case "-o":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("-o requires an argument")
			}
			o.output = args[i+1]
			i++
		case "--audit-large", "--large-kb":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s requires an argument", a)
			}
			var v int64
			if _, err := fmt.Sscanf(args[i+1], "%d", &v); err == nil {
				o.auditLargeKb = v
			} else {
				return nil, fmt.Errorf("%s requires a numerical argument - %w", a, err)
			}
			i++
		default:
			if a != "-" && strings.HasPrefix(a, "-") {
				return nil, fmt.Errorf("unknown flag: %s", a)
			}
			positionals = append(positionals, a)
		}
	}
	if len(positionals) > 2 {
		return nil, fmt.Errorf("unexpected extra argument: %s", positionals[2])
	}
	if o.markdown && o.jsonOut {
		return nil, fmt.Errorf("--markdown and --json are mutually exclusive")
	}
	if len(positionals) >= 1 {
		o.file = positionals[0]
	}
	if len(positionals) == 2 {
		o.file2 = positionals[1]
	}
	// --diff requires two files
	if o.mode == "diff" && o.file2 == "" {
		return nil, fmt.Errorf("--diff requires two HAR files")
	}
	// non-diff modes only accept one file
	if o.mode != "diff" && o.mode != "fields" && o.file2 != "" {
		return nil, fmt.Errorf("unexpected extra argument: %s", o.file2)
	}
	if o.mode == "audit" && o.auditLargeKb == 0 {
		o.auditLargeKb = 150
	}
	return o, nil
}

func printUsage() {
	fmt.Println(`harboring — HTTP Archive Reader In Go

Usage:
  harboring <file.har>              Open TUI viewer (default)
  harboring --view <file.har>       Open TUI viewer
  harboring --dump <file.har>       Dump formatted JSON to stdout
  harboring --scrub <file.har>      Scrub content sections
  harboring --shrink <file.har>     Minify JSON (remove whitespace)
  harboring --expand <file.har>     Pretty-print JSON
  harboring --stats <file.har>      Show summary statistics
  harboring --perf <file.har>       Generate performance report
  harboring --cache <file.har>      Show cache summary
  harboring --security <file.har>   Show security headers analysis
  harboring --trackers <file.har>   Show third-party tracker analysis
  harboring --priority <file.har>   Show requests with changed resource priority
  harboring --validate <file.har>   Validate HAR structure
  harboring --fields <fields> <file.har>  Output HAR with only the specified fields
  harboring --list themes                 List available themes
  harboring --list trackers               List known tracker domains
  harboring --audit images <file.har>     Audit HTML images for optimization issues
  harboring --audit redirects <file.har>  Show redirect chains with timing information
  harboring --help                  Show this help
  harboring --version               Show version

Flags:
  --view, --dump, --scrub, --shrink, --expand, --stats, --perf,
  --cache, --security, --trackers, --priority, --validate, --fields, --diff, --audit
                                    Select the operation (view is the default;
                                    exactly one may be given)
  --theme <name|path>               Theme override (view mode)
  --trackers-file <path>            Load additional tracker domains from a TOML file
                                    (used with --trackers)
  --list <themes|trackers>          List built-in themes or known tracker domains
  --markdown, -m                    Render reports as Markdown
  --json, -j                        Render reports as JSON
  -o <file>                         Output file (--scrub/--shrink/--expand/--stats/--perf/--cache/--security/--trackers/--priority/--fields)
  --audit-large-kb <n>              Large PNG threshold in KB for --audit images (default 150)
  --help, -h                        Show this help
  --version, -v                     Show version

Non-interactive modes (--dump, --scrub, --shrink, --expand, --stats, --perf,
--cache, --security, --trackers, --priority, --validate, --fields) accept a file path,
an explicit "-" for stdin, or piped input with no argument, and write to
stdout unless -o is given.

  --fields takes a comma-separated list of dot-path field names
  (e.g. timings.wait,request.url,response._transferSize). Paths are
  relative to each entry; matching objects keep only the named children.`)
}

func cmdView(o *options) {
	if o.file == "" {
		fmt.Fprintln(os.Stderr, "error: no HAR file specified")
		os.Exit(1)
	}

	root, err := har.ParseFile(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	th := theme.New()
	th.LoadStartup(o.theme)

	p := tea.NewProgram(tui.NewApp(root, o.file, th))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdDiff(o *options) {
	if o.file == "" || o.file2 == "" {
		fmt.Fprintln(os.Stderr, "error: --diff requires two HAR files")
		os.Exit(1)
	}

	left, err := har.ParseFile(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	right, err := har.ParseFile(o.file2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	th := theme.New()
	th.LoadStartup(o.theme)

	p := tea.NewProgram(tui.NewDiffApp(left, right, o.file, o.file2, th))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdDump(o *options) {
	root, err := loadRoot(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	data, err := har.ExportJSON(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: export: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(data))
}

func cmdFields(o *options) {
	root, err := loadRoot(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	paths := strings.Split(o.fields, ",")
	projected := har.ProjectFilters(root, paths)

	data, err := har.ExportJSON(projected)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: export: %v\n", err)
		os.Exit(1)
	}

	if o.output == "" {
		fmt.Print(string(data))
		return
	}

	if err := har.WriteFileAuto(o.output, data); err != nil {
		fmt.Fprintf(os.Stderr, "error: write output: %v\n", err)
		os.Exit(1)
	}
}

func cmdScrub(o *options) {
	root, err := loadRoot(o.file)
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

	if o.output == "" {
		fmt.Print(string(data))
		return
	}

	if err := har.WriteFileAuto(o.output, data); err != nil {
		fmt.Fprintf(os.Stderr, "error: write output: %v\n", err)
		os.Exit(1)
	}
}

func cmdShrink(o *options) {
	if o.output == "" {
		data, err := readInputBytes(o.file)
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

	if o.file == "" || o.file == "-" {
		data, err := readInputBytes(o.file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		minified, err := har.MinifyJSON(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := har.WriteFileAuto(o.output, minified); err != nil {
			fmt.Fprintf(os.Stderr, "error: write output: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := har.MinifyFile(o.file, o.output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdExpand(o *options) {
	if o.output == "" {
		data, err := readInputBytes(o.file)
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

	if o.file == "" || o.file == "-" {
		data, err := readInputBytes(o.file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		expanded, err := har.ExpandJSON(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := har.WriteFileAuto(o.output, expanded); err != nil {
			fmt.Fprintf(os.Stderr, "error: write output: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := har.ExpandFile(o.file, o.output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdStats(o *options) {
	details, err := loadDetails(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if o.jsonOut {
		data, err := report.StatsJSONReport(details)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		writeReport(data, o)
		return
	}
	writeReport([]byte(report.StatsReport(details, o.markdown)), o)
}

func cmdPerf(o *options) {
	details, err := loadDetails(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if o.jsonOut {
		data, err := report.PerfJSONReport(details)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		writeReport(data, o)
		return
	}
	writeReport([]byte(report.PerfReport(details, o.markdown)), o)
}

func cmdCache(o *options) {
	details, err := loadDetails(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if o.jsonOut {
		data, err := report.CacheJSONReport(details)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		writeReport(data, o)
		return
	}
	writeReport([]byte(report.CacheReport(details, o.markdown)), o)
}

func cmdSecurity(o *options) {
	details, err := loadDetails(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if o.jsonOut {
		data, err := report.SecurityJSONReport(details)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		writeReport(data, o)
		return
	}
	writeReport([]byte(report.SecurityReport(details, o.markdown)), o)
}

func cmdTrackers(o *options) {
	details, err := loadDetails(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if o.trackersFile != "" {
		if err := report.LoadTrackersFile(o.trackersFile); err != nil {
			fmt.Fprintf(os.Stderr, "error: --trackers-file: %v\n", err)
			os.Exit(1)
		}
	}

	if o.jsonOut {
		data, err := report.TrackersJSONReport(details)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		writeReport(data, o)
		return
	}
	writeReport([]byte(report.TrackersReport(details, o.markdown)), o)
}

func cmdPriority(o *options) {
	details, err := loadDetails(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if o.jsonOut {
		data, err := report.PriorityJSONReport(details)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		writeReport(data, o)
		return
	}
	writeReport([]byte(report.PriorityReport(details, o.markdown)), o)
}

// listThemes prints the built-in theme names.
func listThemes() {
	th := theme.New()
	fmt.Println("Built-in themes:")
	for _, name := range th.Names() {
		fmt.Printf("  %s\n", name)
	}
}

// listTrackers prints the effective tracker domains grouped by category.
func listTrackers() {
	byCat := map[report.TrackerCategory][]string{}
	for _, t := range report.KnownTrackers() {
		byCat[t.Category] = append(byCat[t.Category], t.Domain)
	}
	categories := []report.TrackerCategory{
		report.TrackerAnalytics,
		report.TrackerAdvertising,
		report.TrackerSocial,
		report.TrackerFingerprinting,
		report.TrackerOther,
	}
	fmt.Println("Known tracker domains:")
	for _, cat := range categories {
		domains := byCat[cat]
		if len(domains) == 0 {
			continue
		}
		fmt.Printf("  %s:\n", cat)
		for _, d := range domains {
			fmt.Printf("    %s\n", d)
		}
	}
}

// writeReport writes data to -o if set, otherwise to stdout.
func writeReport(data []byte, o *options) {
	if o.output != "" {
		// #nosec G703 - Path is intentionally user-defined
		if err := os.WriteFile(o.output, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	os.Stdout.Write(data)
}

func cmdValidate(o *options) {
	root, err := loadRoot(o.file)
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

func cmdAudit(o *options) {
	root, err := loadRoot(o.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	entries := har.GetEntries(root)
	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "error: no entries found\n")
		os.Exit(1)
	}
	details := make([]*har.EntryDetails, len(entries))
	for i, e := range entries {
		details[i] = har.GetEntryDetails(e)
	}

	if o.audit == "redirects" {
		if o.jsonOut {
			data, err := report.AuditRedirectsJSONReport(root, details)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			writeReport(data, o)
			return
		}
		out := report.AuditRedirectsReport(root, details, o.markdown)
		writeReport([]byte(out), o)
		return
	}

	if o.jsonOut {
		data, err := report.AuditImagesJSONReport(root, details, o.auditLargeKb)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		writeReport(data, o)
		return
	}
	out := report.AuditImagesReport(root, details, o.auditLargeKb, o.markdown)
	writeReport([]byte(out), o)
}

// stdinIsTerminal reports whether os.Stdin is an interactive terminal (as
// opposed to a pipe or redirected file). Used to avoid blocking on stdin when a
// non-interactive mode is run with no file argument.
func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// loadRoot parses a HAR from file (a path), or from stdin when file is "" or
// "-". If file is empty and stdin is a terminal, it returns an error instead of
// blocking on interactive input.
func loadRoot(file string) (*har.Node, error) {
	if file != "-" {
		if file != "" {
			return har.ParseFile(file)
		}
		if stdinIsTerminal() {
			return nil, fmt.Errorf("no HAR file specified (pipe a HAR via stdin or pass a file path)")
		}
	}
	return har.ParseReader(os.Stdin)
}

// readInputBytes returns the raw input bytes from file (a path) or from stdin
// when file is "" or "-".
func readInputBytes(file string) ([]byte, error) {
	if file != "-" {
		if file != "" {
			return har.ReadFileLimited(file)
		}
		if stdinIsTerminal() {
			return nil, fmt.Errorf("no HAR file specified (pipe a HAR via stdin or pass a file path)")
		}
	}
	return har.ReadAllLimited(os.Stdin)
}

// loadDetails parses the HAR at file (path, "-" or piped stdin) and returns the
// per-entry details, erroring if there are no entries.
func loadDetails(file string) ([]*har.EntryDetails, error) {
	root, err := loadRoot(file)
	if err != nil {
		return nil, err
	}
	entries := har.GetEntries(root)
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found")
	}
	details := make([]*har.EntryDetails, len(entries))
	for i, e := range entries {
		details[i] = har.GetEntryDetails(e)
	}
	return details, nil
}

func findChild(n *har.Node, key string) *har.Node {
	for _, child := range n.Children {
		if child.Key == key {
			return child
		}
	}
	return nil
}
