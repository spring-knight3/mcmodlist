// Package cli implements the mcmodlist command line interface.
package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"mcmodlist/cli/render"
	"mcmodlist/core"
	instancepkg "mcmodlist/core/instance"
	"mcmodlist/core/model"
)

// Version is the CLI version string. It is a variable so release builds can
// inject the package version with -ldflags "-X mcmodlist/cli.Version=...".
var Version = "dev"

var formats = map[string]bool{
	"pretty":   true,
	"markdown": true,
	"csv":      true,
	"json":     true,
	"html":     true,
	"text":     true,
}

var launchers = map[string]bool{
	"prism":      true,
	"multimc":    true,
	"curseforge": true,
	"modrinth":   true,
	"atlauncher": true,
	"gdlauncher": true,
}

var columnNames = map[string]string{
	"name":    "Name",
	"id":      "ID",
	"version": "Version",
	"author":  "Authors",
	"authors": "Authors",
	"jar":     "Jar",
	"loader":  "Loader",
	"side":    "Side",
	"status":  "Status",
}

var (
	isTTY                   = isTerminal
	isTTYInput              = isTerminalReader
	clipboard     Clipboard = execClipboard{}
	listInstances           = core.ListInstances
)

// Clipboard copies text to the system clipboard.
type Clipboard interface {
	Copy(text string) error
}

// Run executes the CLI and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunIO(args, os.Stdin, stdout, stderr)
}

// RunIO executes the CLI with an explicit stdin for interactive prompts.
func RunIO(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flagArgs, positional := splitArgs(args)
	if err := validateFlagNames(flagArgs); err != nil {
		fmt.Fprintf(stderr, "error: %v\nrun 'mcmodlist --help' for usage\n", err)
		return 2
	}
	cfg, cfgWarning := loadConfig()
	if cfgWarning != "" {
		fmt.Fprintln(stderr, cfgWarning)
	}
	defaultFormat := cfg.Format
	if defaultFormat == "" {
		defaultFormat = "pretty"
	}
	defaultColor := cfg.Color
	if defaultColor == "" {
		defaultColor = "auto"
	}

	fs := flag.NewFlagSet("mcmodlist", flag.ContinueOnError)
	fs.SetOutput(stderr)

	instanceShort := fs.String("i", "", "instance directory")
	instanceLong := fs.String("instance", "", "instance directory")
	formatShort := fs.String("f", "", "output format")
	formatLong := fs.String("format", defaultFormat, "output format")
	outputShort := fs.String("o", "", "write output to file")
	outputLong := fs.String("output", "", "write output to file")
	force := fs.Bool("force", false, "overwrite an existing output file")
	noSummary := fs.Bool("no-summary", false, "omit the instance summary")
	summaryOnly := fs.Bool("summary-only", false, "print only the instance summary")
	noHeader := fs.Bool("no-header", false, "omit headers for data-only output")
	color := fs.String("color", defaultColor, "color mode: auto, always, or never")
	noColor := fs.Bool("no-color", false, "disable colour")
	verboseShort := fs.Bool("v", false, "print extra diagnostics to stderr")
	verboseLong := fs.Bool("verbose", false, "print extra diagnostics to stderr")
	quietShort := fs.Bool("q", false, "suppress warnings")
	quietLong := fs.Bool("quiet", false, "suppress warnings")
	helpShort := fs.Bool("h", false, "show help")
	helpLong := fs.Bool("help", false, "show help")
	versionShort := fs.Bool("V", false, "show version")
	versionLong := fs.Bool("version", false, "show version")

	launcher := fs.String("launcher", "", "filter --list-instances by launcher")
	listFlag := fs.Bool("list-instances", false, "list known instances and exit")
	includeDisabled := fs.Bool("include-disabled", false, "include disabled mods")
	includeNested := fs.Bool("include-nested", false, "include mods nested inside jars")
	recursive := fs.Bool("recursive", false, "scan subdirectories of the mods folder")
	jobs := fs.Int("jobs", 0, "jar scan worker count")
	columnsFlag := fs.String("columns", cfg.Columns, "comma-separated columns")
	sortSpec := fs.String("sort", "", "sort by name,id,version,author,jar,loader,side,status (- reverses)")
	groupBy := fs.String("group-by", "", "group output by loader or side")
	wide := fs.Bool("wide", false, "do not truncate table columns")
	truncate := fs.Int("truncate", 0, "truncate table cells to N characters")
	copyFlag := fs.Bool("copy", false, "copy rendered output to the clipboard")
	search := fs.String("search", "", "filter mods by text")
	loaderFilter := fs.String("loader", "", "filter mods by loader")
	sideFilter := fs.String("side", "", "filter mods by side")
	authorFilter := fs.String("author", "", "filter mods by author")
	unrecognised := fs.Bool("unrecognised", false, "show only unrecognised jars")
	checkFlag := fs.Bool("check", false, "check for problems and exit 3 if found")
	diffPath := fs.String("diff", "", "compare with another instance")
	guiFlag := fs.Bool("gui", false, "launch the GUI if installed")

	if err := fs.Parse(flagArgs); err != nil {
		fmt.Fprintln(stderr, "run 'mcmodlist --help' for usage")
		return 2
	}
	if *helpShort || *helpLong {
		helpInstance := firstNonEmpty(*instanceShort, *instanceLong)
		if helpInstance == "" && len(positional) == 1 {
			helpInstance = positional[0]
		}
		if helpInstance != "" {
			colorMode := *color
			if *noColor {
				colorMode = "never"
			}
			printInstanceCheck(stdout, helpInstance, colorEnabled(colorMode, stdout))
		}
		usage(stdout)
		return 0
	}
	if *versionShort || *versionLong {
		fmt.Fprintf(stdout, "mcmodlist %s\n", Version)
		return 0
	}
	if *guiFlag {
		return launchGUI(stderr)
	}
	if len(positional) > 0 && positional[0] == "completion" {
		if len(positional) != 2 {
			fmt.Fprintln(stderr, "error: usage: mcmodlist completion <bash|zsh|fish|powershell>")
			return 2
		}
		script, err := completionScript(positional[1])
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 2
		}
		fmt.Fprint(stdout, script)
		return 0
	}
	if len(positional) > 1 {
		fmt.Fprintln(stderr, "error: too many arguments; expected at most one instance directory")
		return 2
	}

	format := firstNonEmpty(*formatShort, *formatLong)
	if !formats[strings.ToLower(format)] {
		if suggestion := strings.TrimLeft(suggest(format, formatKeys()), "-"); suggestion != "" {
			fmt.Fprintf(stderr, "error: invalid format %q (want pretty, markdown, csv, json, html, or text); did you mean %q?\n", format, suggestion)
		} else {
			fmt.Fprintf(stderr, "error: invalid format %q (want pretty, markdown, csv, json, html, or text)\n", format)
		}
		return 2
	}
	columns, err := parseColumns(*columnsFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	if gb := strings.ToLower(strings.TrimSpace(*groupBy)); gb != "" && gb != "loader" && gb != "side" {
		fmt.Fprintf(stderr, "error: invalid --group-by %q (want loader or side)\n", *groupBy)
		return 2
	}
	if *jobs < 0 {
		fmt.Fprintln(stderr, "error: --jobs must be >= 0")
		return 2
	}
	if *truncate < 0 {
		fmt.Fprintln(stderr, "error: --truncate must be >= 0")
		return 2
	}
	if _, err := core.SortMods(nil, *sortSpec); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	launcherFilter := strings.ToLower(strings.TrimSpace(*launcher))
	if launcherFilter != "" && !launchers[launcherFilter] {
		fmt.Fprintf(stderr, "error: invalid --launcher %q (want prism, multimc, curseforge, modrinth, atlauncher, or gdlauncher)\n", *launcher)
		return 2
	}
	if *listFlag {
		return listInstancesCmd(launcherFilter, stdout, stderr)
	}
	if launcherFilter != "" {
		fmt.Fprintln(stderr, "error: --launcher only applies to --list-instances")
		return 2
	}

	output := firstNonEmpty(*outputShort, *outputLong)
	instanceFlag := firstNonEmpty(*instanceShort, *instanceLong)
	positionalInstance := ""
	if len(positional) == 1 {
		positionalInstance = positional[0]
	}
	if instanceFlag != "" && positionalInstance != "" && instanceFlag != positionalInstance {
		fmt.Fprintln(stderr, "error: instance specified twice")
		return 2
	}
	instancePath := instanceFlag
	if instancePath == "" {
		instancePath = positionalInstance
	}
	if instancePath == "" {
		instancePath = "."
		if _, err := instancepkg.Detect(instancePath); err != nil && isTTYInput(stdin) && isTTY(stdout) {
			path, ok := pickInstance(stdin, stdout)
			if !ok {
				return 1
			}
			instancePath = path
		}
	}

	colorMode := *color
	if *noColor {
		colorMode = "never"
	}
	switch colorMode {
	case "auto", "always", "never":
	default:
		fmt.Fprintf(stderr, "error: invalid color mode %q (want auto, always, or never)\n", colorMode)
		return 2
	}
	if *noSummary && *summaryOnly {
		fmt.Fprintln(stderr, "error: --no-summary and --summary-only cannot be used together")
		return 2
	}
	verbose := *verboseShort || *verboseLong
	quiet := *quietShort || *quietLong

	progressShown := false
	var progressMu sync.Mutex
	progress := func(done, total int) {
		progressMu.Lock()
		defer progressMu.Unlock()
		if !isTTY(stderr) || total < 50 {
			return
		}
		progressShown = true
		fmt.Fprintf(stderr, "\rscanning jars %d/%d", done, total)
	}
	report, err := core.Scan(instancePath, core.Options{
		Jobs:          *jobs,
		IncludeNested: *includeNested,
		Recursive:     *recursive,
		Progress:      progress,
	})
	progressMu.Lock()
	shownProgress := progressShown
	progressMu.Unlock()
	if shownProgress {
		fmt.Fprintln(stderr)
	}
	if err != nil {
		printScanError(stderr, err)
		return 1
	}
	if !quiet {
		for _, warning := range report.Warnings {
			printWarning(stderr, warning)
		}
	}
	if verbose {
		fmt.Fprintf(stderr, "mcmodlist: scanned %d mod(s) in %s\n", len(report.Mods), report.Instance.Path)
	}

	checkIssues := []model.Issue{}
	if *checkFlag {
		checkIssues = core.Check(report)
		report.Issues = checkIssues
		for _, issue := range checkIssues {
			printIssue(stderr, issue)
		}
	}

	filterOpts := core.FilterOptions{
		Search:           *search,
		Loader:           *loaderFilter,
		Side:             *sideFilter,
		Author:           *authorFilter,
		UnrecognisedOnly: *unrecognised,
		IncludeDisabled:  *includeDisabled,
	}
	view := core.FilterReport(report, filterOpts)
	if strings.TrimSpace(*sortSpec) != "" {
		sorted, err := core.SortMods(view.Mods, *sortSpec)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 2
		}
		view.Mods = sorted
		view.Counts = core.CountMods(view.Mods)
	}
	skippedDisabled := report.Counts.Disabled - view.Counts.Disabled
	if skippedDisabled < 0 {
		skippedDisabled = 0
	}
	exitCode := 0
	if *checkFlag && len(checkIssues) > 0 {
		exitCode = 3
	}

	renderReport := view
	if *diffPath != "" {
		other, err := core.Scan(*diffPath, core.Options{
			Jobs:          *jobs,
			IncludeNested: *includeNested,
			Recursive:     *recursive,
		})
		if err != nil {
			printScanError(stderr, err)
			return 1
		}
		renderReport = core.Diff(report, other)
		if *checkFlag {
			renderReport.Issues = checkIssues
		}
		renderReport = core.FilterReport(renderReport, filterOpts)
		if strings.TrimSpace(*sortSpec) != "" {
			sorted, err := core.SortMods(renderReport.Mods, *sortSpec)
			if err != nil {
				fmt.Fprintf(stderr, "error: %v\n", err)
				return 2
			}
			renderReport.Mods = sorted
			renderReport.Counts = core.CountMods(renderReport.Mods)
		}
		skippedDisabled = 0
	}

	useColor := colorEnabled(colorMode, stdout)
	out, err := render.Render(format, renderReport, render.Options{
		Color:           useColor,
		NoHeader:        *noHeader,
		NoSummary:       *noSummary,
		SummaryOnly:     *summaryOnly,
		Width:           0,
		Version:         Version,
		Columns:         columns,
		GroupBy:         *groupBy,
		Wide:            *wide,
		Truncate:        *truncate,
		SkippedDisabled: skippedDisabled,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	out = strings.ToValidUTF8(out, "�")
	if *copyFlag {
		if err := clipboard.Copy(out); err != nil {
			fmt.Fprintf(stderr, "error: clipboard unavailable: %v\n", err)
			return 1
		}
		if output == "" || output == "-" {
			return exitCode
		}
	}
	if output == "" || output == "-" {
		_, _ = io.WriteString(stdout, out)
		return exitCode
	}
	if !*force {
		if _, err := os.Stat(output); err == nil {
			fmt.Fprintf(stderr, "error: output file %s exists; use --force to overwrite\n", output)
			return 2
		}
	}
	if err := os.WriteFile(output, []byte(out), 0o644); err != nil {
		fmt.Fprintf(stderr, "error: cannot write %s: %v\n", output, err)
		return 1
	}
	return exitCode
}

func guiBinaryName() string {
	if runtime.GOOS == "windows" {
		return "mcmodlist-gui.exe"
	}
	return "mcmodlist-gui"
}

func launchGUI(stderr io.Writer) int {
	path, err := exec.LookPath(guiBinaryName())
	if err != nil {
		fmt.Fprintf(stderr, "error: mcmodlist GUI is not installed; build it with 'scripts/build-all.sh' or install the 'mcmodlist-gui' package\n")
		return 1
	}
	cmd := exec.Command(path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "error: GUI exited: %v\n", err)
		return 1
	}
	return 0
}

func listInstancesCmd(launcherFilter string, stdout, stderr io.Writer) int {
	instances, err := listInstances()
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot list instances: %v\n", err)
		return 1
	}
	count := 0
	for _, info := range instances {
		if launcherFilter != "" && strings.ToLower(info.Launcher) != launcherFilter {
			continue
		}
		fmt.Fprintf(stdout, "%s\t%s\n", info.Name, info.Path)
		count++
	}
	if count == 0 {
		fmt.Fprintln(stderr, "no instances found")
	}
	return 0
}

func pickInstance(stdin io.Reader, stdout io.Writer) (string, bool) {
	instances, _ := listInstances()
	reader := bufio.NewReader(stdin)
	if len(instances) > 0 {
		fmt.Fprintln(stdout, "Select an instance:")
		for i, info := range instances {
			fmt.Fprintf(stdout, "%2d) %s (%s)\n", i+1, info.Name, info.Path)
		}
		fmt.Fprint(stdout, "Enter number or path [q]: ")
	} else {
		fmt.Fprint(stdout, "Instance path: ")
	}
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", false
	}
	line = strings.TrimSpace(line)
	if line == "" || strings.EqualFold(line, "q") {
		return "", false
	}
	if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(instances) {
		return instances[n-1].Path, true
	}
	return line, true
}

var valueFlagNames = map[string]bool{
	"i": true, "instance": true,
	"f": true, "format": true,
	"o": true, "output": true,
	"color":    true,
	"launcher": true, "jobs": true, "columns": true, "sort": true,
	"group-by": true, "truncate": true,
	"search": true, "loader": true, "side": true, "author": true,
	"diff": true,
}

func splitArgs(args []string) (flagArgs, positional []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			name := strings.TrimLeft(arg, "-")
			if eq := strings.Index(name, "="); eq >= 0 {
				name = name[:eq]
			}
			flagArgs = append(flagArgs, arg)
			if valueFlagNames[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		positional = append(positional, arg)
	}
	return flagArgs, positional
}

func knownFlags() map[string]bool {
	return map[string]bool{
		"i": true, "instance": true,
		"f": true, "format": true,
		"o": true, "output": true,
		"force": true, "no-summary": true, "summary-only": true,
		"no-header": true, "color": true, "no-color": true,
		"v": true, "verbose": true, "q": true, "quiet": true,
		"h": true, "help": true, "V": true, "version": true,
		"launcher": true, "list-instances": true,
		"include-disabled": true, "include-nested": true, "recursive": true,
		"jobs": true, "columns": true, "sort": true, "group-by": true,
		"wide": true, "truncate": true, "copy": true,
		"search": true, "loader": true, "side": true, "author": true,
		"unrecognised": true, "check": true, "diff": true, "gui": true,
	}
}

func validateFlagNames(flagArgs []string) error {
	known := knownFlags()
	for i := 0; i < len(flagArgs); i++ {
		arg := flagArgs[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		hasValue := strings.Contains(arg, "=")
		if eq := strings.Index(name, "="); eq >= 0 {
			name = name[:eq]
		}
		if name == "" || known[name] {
			if valueFlagNames[name] && !hasValue {
				i++
			}
			continue
		}
		display := "--" + name
		if len(name) == 1 {
			display = "-" + name
		}
		if suggestion := suggest(name, flagKeys()); suggestion != "" {
			return fmt.Errorf("unknown flag %s; did you mean %s?", display, suggestion)
		}
		return fmt.Errorf("unknown flag %s", display)
	}
	return nil
}

func flagKeys() []string {
	keys := make([]string, 0, len(knownFlags()))
	for key := range knownFlags() {
		keys = append(keys, key)
	}
	return keys
}

func formatKeys() []string {
	keys := make([]string, 0, len(formats))
	for key := range formats {
		keys = append(keys, key)
	}
	return keys
}

func parseColumns(spec string) ([]string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	var out []string
	for _, part := range strings.Split(spec, ",") {
		key := strings.ToLower(strings.TrimSpace(part))
		if key == "" {
			continue
		}
		name, ok := columnNames[key]
		if !ok {
			return nil, fmt.Errorf("unknown column %q (want name,id,version,author,jar,loader,side,status)", part)
		}
		out = append(out, name)
	}
	return out, nil
}

func suggest(input string, candidates []string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	best := ""
	bestDist := 1 << 30
	for _, candidate := range candidates {
		dist := levenshtein(input, strings.ToLower(candidate))
		if dist < bestDist {
			bestDist = dist
			best = candidate
		}
	}
	if bestDist <= 2 {
		if len(best) == 1 {
			return "-" + best
		}
		return "--" + best
	}
	return ""
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func colorEnabled(mode string, stdout io.Writer) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default:
		if _, ok := os.LookupEnv("NO_COLOR"); ok {
			return false
		}
		return isTTY(stdout)
	}
}

func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func isTerminalReader(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func printInstanceCheck(w io.Writer, path string, useColor bool) {
	_, err := instancepkg.Detect(path)
	if err == nil {
		if useColor {
			fmt.Fprintln(w, "\x1b[32mThis looks like a valid instance.\x1b[0m")
		} else {
			fmt.Fprintln(w, "This looks like a valid instance.")
		}
		return
	}
	if useColor {
		fmt.Fprintln(w, "\x1b[31mThis does not look like a valid instance.\x1b[0m")
	} else {
		fmt.Fprintln(w, "This does not look like a valid instance.")
	}
}

func printScanError(stderr io.Writer, err error) {
	var detectErr *instancepkg.Error
	if errors.As(err, &detectErr) && detectErr.Kind == instancepkg.KindModsFolder && detectErr.Suggestion != "" {
		fmt.Fprintf(stderr, "error: %v\ntry: mcmodlist %s\n", err, detectErr.Suggestion)
		return
	}
	fmt.Fprintf(stderr, "error: %v\n", err)
}

func printIssue(stderr io.Writer, issue model.Issue) {
	if issue.Mod != "" {
		fmt.Fprintf(stderr, "issue[%s] %s: %s\n", issue.Type, issue.Mod, issue.Message)
		return
	}
	fmt.Fprintf(stderr, "issue[%s]: %s\n", issue.Type, issue.Message)
}

func printWarning(stderr io.Writer, warning model.Warning) {
	if warning.Mod != "" {
		fmt.Fprintf(stderr, "warning[%s] %s: %s\n", warning.Type, warning.Mod, warning.Message)
		return
	}
	fmt.Fprintf(stderr, "warning[%s]: %s\n", warning.Type, warning.Message)
}

func usage(w io.Writer) {
	fmt.Fprint(w, `mcmodlist - list the mods in a Minecraft instance

Usage:
  mcmodlist [instance] [flags]
  mcmodlist -i <instance> [flags]
  mcmodlist --list-instances [--launcher <name>]

Instance flags:
  -i, --instance <path>   instance directory (default "." or picker on a TTY)
      --launcher <name>   filter --list-instances (prism, multimc, curseforge,
                          modrinth, atlauncher, gdlauncher)
      --list-instances    list known instances and exit
      --include-disabled  include disabled mods (hidden by default)
      --include-nested    include mods nested inside jars
      --recursive         scan subdirectories of the mods folder
      --jobs <n>          jar scan worker count

Output flags:
  -f, --format <format>   pretty, markdown, csv, json, html, or text (default pretty)
  -o, --output <path>     write output to a file ("-" for stdout)
      --force             overwrite an existing output file
      --columns <list>    comma-separated columns: name,id,version,author,jar,
                          loader,side,status
      --sort <spec>       comma-separated sort keys; prefix '-' reverses
      --group-by <key>    group by loader or side
      --wide              do not truncate table columns
      --truncate <n>      truncate table cells to n characters
      --copy              copy rendered output to the clipboard
      --no-summary        omit the instance summary
      --summary-only      print only the instance summary
      --no-header         omit headers for data-only output
      --color <mode>      auto, always, or never (default auto)
      --no-color          disable colour

Filter flags:
      --search <text>     filter mods by name/id/version/author/jar text
      --loader <name>     filter by loader (fabric, quilt, forge, neoforge,
                          connector, or loader/layer)
      --side <name>       filter by side (client, server, both)
      --author <name>     filter by author substring
      --unrecognised      show only unrecognised jars

Diagnostics and comparison:
      --check             check for duplicate ids, loader mismatches, missing
                          dependencies, and wrong Minecraft versions
      --diff <path>       compare this instance with another instance

Other flags:
  -v, --verbose           print extra diagnostics to stderr
  -q, --quiet             suppress warnings
  -h, --help              show this help
  -V, --version           show version
      --gui               launch the GUI if installed

Commands:
  mcmodlist completion <bash|zsh|fish|powershell>
                          print a shell completion script

Config:
  ~/.config/mcmodlist/config.toml can set format, columns, and color.
  Command-line flags override config values.

Notes:
  Loader counts and totals reflect the mods shown after filtering.
  Disabled mods are hidden unless --include-disabled is used.
  --check exits 3 when it finds problems, 0 when the instance is clean.

Examples:
  mcmodlist
  mcmodlist ~/Games/Prism/instances/1.20.1-Fabric
  mcmodlist /path/to/instance --format markdown --copy
  mcmodlist /path/to/instance --format json --output mods.json --force
  mcmodlist /path/to/instance --search sodium --sort -version
  mcmodlist /path/to/instance --check
  mcmodlist /path/to/instance --diff /path/to/other-instance
  mcmodlist --list-instances --launcher prism
  mcmodlist completion bash
`)
}
