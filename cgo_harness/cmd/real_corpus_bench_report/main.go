package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var benchmarkSuffixRE = regexp.MustCompile(`-\d+$`)

type sample struct {
	Suite      string             `json:"suite"`
	Language   string             `json:"language"`
	Backend    string             `json:"backend"`
	Iterations int64              `json:"iterations"`
	Metrics    map[string]float64 `json:"metrics"`
	Source     string             `json:"source,omitempty"`
}

type metricSummary struct {
	Samples         int     `json:"samples"`
	Median          float64 `json:"median"`
	Mean            float64 `json:"mean"`
	Min             float64 `json:"min"`
	Max             float64 `json:"max"`
	StdDev          float64 `json:"stddev,omitempty"`
	RelStdDev       float64 `json:"rel_stddev,omitempty"`
	MedianAbsDev    float64 `json:"median_abs_dev,omitempty"`
	RelMedianAbsDev float64 `json:"rel_median_abs_dev,omitempty"`
}

type benchmarkSummary struct {
	Suite      string                   `json:"suite"`
	Language   string                   `json:"language"`
	Backend    string                   `json:"backend"`
	Iterations metricSummary            `json:"iterations"`
	Metrics    map[string]metricSummary `json:"metrics"`
}

type languageReport struct {
	Language               string                        `json:"language"`
	FullRatio              float64                       `json:"full_ratio,omitempty"`
	EditRatio              float64                       `json:"edit_ratio,omitempty"`
	NoEditRatio            float64                       `json:"noedit_ratio,omitempty"`
	WorstRatio             float64                       `json:"worst_ratio,omitempty"`
	FullGoNanos            float64                       `json:"full_go_ns,omitempty"`
	FullCNanos             float64                       `json:"full_c_ns,omitempty"`
	EditGoNanos            float64                       `json:"edit_go_ns,omitempty"`
	EditCNanos             float64                       `json:"edit_c_ns,omitempty"`
	NoEditGoNanos          float64                       `json:"noedit_go_ns,omitempty"`
	NoEditCNanos           float64                       `json:"noedit_c_ns,omitempty"`
	MinNSOpSamples         int                           `json:"min_ns_op_samples,omitempty"`
	MaxNSOpRelStdDev       float64                       `json:"max_ns_op_rel_stddev,omitempty"`
	MaxNSOpRelMedianAbsDev float64                       `json:"max_ns_op_rel_median_abs_dev,omitempty"`
	MaxRSSKB               int64                         `json:"max_rss_kb,omitempty"`
	OOMKilled              bool                          `json:"oom_killed,omitempty"`
	QualityNotes           []string                      `json:"quality_notes,omitempty"`
	TopAttribution         []attributionBucket           `json:"top_attribution,omitempty"`
	GoSuiteMetrics         map[string]map[string]float64 `json:"go_suite_metrics,omitempty"`
	CSuiteMetrics          map[string]map[string]float64 `json:"c_suite_metrics,omitempty"`
}

type attributionBucket struct {
	Suite string  `json:"suite"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type report struct {
	GeneratedAt string             `json:"generated_at"`
	Inputs      []string           `json:"inputs"`
	Samples     int                `json:"samples"`
	Runs        []runResource      `json:"runs,omitempty"`
	Benchmarks  []benchmarkSummary `json:"benchmarks"`
	Languages   []languageReport   `json:"languages"`
}

type runResource struct {
	SourceDir     string `json:"source_dir"`
	Language      string `json:"language,omitempty"`
	MaxRSSKB      int64  `json:"max_rss_kb,omitempty"`
	OOMKilled     *bool  `json:"oom_killed,omitempty"`
	ExitCode      *int   `json:"exit_code,omitempty"`
	Memory        string `json:"memory,omitempty"`
	GOMEMLimit    string `json:"gomemlimit,omitempty"`
	AllowMismatch *bool  `json:"allow_mismatch,omitempty"`
	SkipMismatch  *bool  `json:"skip_mismatch,omitempty"`
}

type key struct {
	suite    string
	language string
	backend  string
}

func main() {
	var (
		inputsCSV string
		outJSON   string
		outMD     string
	)
	flag.StringVar(&inputsCSV, "input", "", "comma-separated benchmark log files or directories; positional args are also accepted")
	flag.StringVar(&outJSON, "out-json", "real_corpus_bench_report.json", "JSON report path")
	flag.StringVar(&outMD, "out-md", "REAL_CORPUS_BENCH_REPORT.md", "Markdown report path")
	flag.Parse()

	inputs := parseInputList(inputsCSV)
	inputs = append(inputs, flag.Args()...)
	if len(inputs) == 0 {
		fatalf("provide at least one benchmark log via -input or positional args")
	}
	files, err := expandInputs(inputs)
	if err != nil {
		fatalf("expand inputs: %v", err)
	}
	if len(files) == 0 {
		fatalf("no benchmark log files selected")
	}

	samples, err := parseBenchmarkFiles(files)
	if err != nil {
		fatalf("parse benchmark logs: %v", err)
	}
	if len(samples) == 0 {
		fatalf("no BenchmarkParityRealCorpusParse lines found")
	}
	resources, err := parseRunResources(files)
	if err != nil {
		fatalf("parse run resources: %v", err)
	}

	r := buildReport(files, samples, resources)
	if err := writeJSON(outJSON, r); err != nil {
		fatalf("write %s: %v", outJSON, err)
	}
	if err := writeMarkdown(outMD, r); err != nil {
		fatalf("write %s: %v", outMD, err)
	}
	fmt.Printf("wrote report json: %s\n", outJSON)
	fmt.Printf("wrote report markdown: %s\n", outMD)
	fmt.Printf("samples: %d languages: %d\n", r.Samples, len(r.Languages))
}

func parseInputList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func expandInputs(inputs []string) ([]string, error) {
	seen := map[string]bool{}
	var files []string
	for _, input := range inputs {
		st, err := os.Stat(input)
		if err != nil {
			return nil, err
		}
		if !st.IsDir() {
			if !seen[input] {
				seen[input] = true
				files = append(files, input)
			}
			continue
		}
		err = filepath.WalkDir(input, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".log", ".txt", ".out":
			default:
				return nil
			}
			if !seen[path] {
				seen[path] = true
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func parseBenchmarkFiles(files []string) ([]sample, error) {
	var samples []sample
	for _, path := range files {
		fileSamples, err := parseBenchmarkFile(path)
		if err != nil {
			return nil, err
		}
		samples = append(samples, fileSamples...)
	}
	return samples, nil
}

func parseRunResources(files []string) ([]runResource, error) {
	byDir := map[string]*runResource{}
	for _, path := range files {
		base := filepath.Base(path)
		if base != "container.log" && base != "metadata.txt" {
			continue
		}
		dir := filepath.Dir(path)
		resource := byDir[dir]
		if resource == nil {
			resource = &runResource{
				SourceDir: dir,
				Language:  inferLanguageFromRunDir(dir),
			}
			byDir[dir] = resource
		}
		switch base {
		case "container.log":
			if err := parseContainerLogResource(path, resource); err != nil {
				return nil, err
			}
		case "metadata.txt":
			if err := parseMetadataResource(path, resource); err != nil {
				return nil, err
			}
		}
	}
	resources := make([]runResource, 0, len(byDir))
	for _, resource := range byDir {
		resources = append(resources, *resource)
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].SourceDir < resources[j].SourceDir
	})
	return resources, nil
}

func inferLanguageFromRunDir(path string) string {
	base := filepath.Base(path)
	const marker = "real-corpus-bench-"
	idx := strings.LastIndex(base, marker)
	if idx < 0 {
		return ""
	}
	return base[idx+len(marker):]
}

func parseContainerLogResource(path string, resource *runResource) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	const rssPrefix = "Maximum resident set size (kbytes):"
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, rssPrefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, rssPrefix))
		rssKB, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%s: parse max RSS %q: %w", path, value, err)
		}
		if rssKB > resource.MaxRSSKB {
			resource.MaxRSSKB = rssKB
		}
	}
	return scanner.Err()
}

func parseMetadataResource(path string, resource *runResource) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "exit_code":
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("%s: parse exit_code %q: %w", path, value, err)
			}
			resource.ExitCode = &n
		case "oom_killed":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("%s: parse oom_killed %q: %w", path, value, err)
			}
			resource.OOMKilled = &b
		case "memory":
			resource.Memory = value
		case "gomemlimit":
			resource.GOMEMLimit = value
		case "allow_mismatch":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("%s: parse allow_mismatch %q: %w", path, value, err)
			}
			resource.AllowMismatch = &b
		case "skip_mismatch":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("%s: parse skip_mismatch %q: %w", path, value, err)
			}
			resource.SkipMismatch = &b
		case "command":
			if b, ok, err := parseCommandEnvBool(value, "GTS_REAL_CORPUS_BENCH_ALLOW_MISMATCH"); err != nil {
				return fmt.Errorf("%s: parse command allow mismatch: %w", path, err)
			} else if ok {
				resource.AllowMismatch = &b
			}
			if b, ok, err := parseCommandEnvBool(value, "GTS_REAL_CORPUS_BENCH_SKIP_MISMATCH"); err != nil {
				return fmt.Errorf("%s: parse command skip mismatch: %w", path, err)
			} else if ok {
				resource.SkipMismatch = &b
			}
		}
	}
	return scanner.Err()
}

func parseCommandEnvBool(command, name string) (bool, bool, error) {
	prefix := name + "="
	for _, field := range strings.Fields(command) {
		field = strings.Trim(field, `"'`)
		if !strings.HasPrefix(field, prefix) {
			continue
		}
		raw := strings.Trim(strings.TrimPrefix(field, prefix), `"'`)
		b, err := strconv.ParseBool(raw)
		return b, true, err
	}
	return false, false, nil
}

func parseBenchmarkFile(path string) ([]sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var samples []sample
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "BenchmarkParityRealCorpusParse") {
			continue
		}
		s, ok, err := parseBenchmarkLine(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		if !ok {
			continue
		}
		s.Source = path
		samples = append(samples, s)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return samples, nil
}

func parseBenchmarkLine(line string) (sample, bool, error) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return sample{}, false, nil
	}
	suite, lang, backend, ok := parseBenchmarkName(fields[0])
	if !ok {
		return sample{}, false, nil
	}
	iterations, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return sample{}, false, fmt.Errorf("parse iteration count %q: %w", fields[1], err)
	}
	metrics := make(map[string]float64)
	for i := 2; i+1 < len(fields); i += 2 {
		value, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			continue
		}
		unit := fields[i+1]
		metrics[unit] = value
	}
	return sample{Suite: suite, Language: lang, Backend: backend, Iterations: iterations, Metrics: metrics}, true, nil
}

func parseBenchmarkName(name string) (suite, language, backend string, ok bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 3 {
		return "", "", "", false
	}
	const prefix = "BenchmarkParityRealCorpusParse"
	if !strings.HasPrefix(parts[0], prefix) {
		return "", "", "", false
	}
	suite = strings.TrimPrefix(parts[0], prefix)
	backend = benchmarkSuffixRE.ReplaceAllString(parts[2], "")
	return suite, parts[1], backend, true
}

func buildReport(inputs []string, samples []sample, resources []runResource) report {
	grouped := make(map[key][]sample)
	for _, s := range samples {
		k := key{suite: s.Suite, language: s.Language, backend: s.Backend}
		grouped[k] = append(grouped[k], s)
	}

	summaries := make([]benchmarkSummary, 0, len(grouped))
	for k, group := range grouped {
		summaries = append(summaries, benchmarkSummary{
			Suite:      k.suite,
			Language:   k.language,
			Backend:    k.backend,
			Iterations: summarizeIterations(group),
			Metrics:    summarizeMetrics(group),
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		a, b := summaries[i], summaries[j]
		if a.Language != b.Language {
			return a.Language < b.Language
		}
		if a.Suite != b.Suite {
			return a.Suite < b.Suite
		}
		return a.Backend < b.Backend
	})

	return report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Inputs:      inputs,
		Samples:     len(samples),
		Runs:        resources,
		Benchmarks:  summaries,
		Languages:   buildLanguageReports(summaries, resources),
	}
}

func summarizeMetrics(samples []sample) map[string]metricSummary {
	valuesByName := make(map[string][]float64)
	for _, s := range samples {
		for name, value := range s.Metrics {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				continue
			}
			valuesByName[name] = append(valuesByName[name], value)
		}
	}
	out := make(map[string]metricSummary, len(valuesByName))
	for name, values := range valuesByName {
		out[name] = summarizeValues(values)
	}
	return out
}

func summarizeIterations(samples []sample) metricSummary {
	values := make([]float64, 0, len(samples))
	for _, s := range samples {
		if s.Iterations > 0 {
			values = append(values, float64(s.Iterations))
		}
	}
	return summarizeValues(values)
}

func summarizeValues(values []float64) metricSummary {
	if len(values) == 0 {
		return metricSummary{}
	}
	sort.Float64s(values)
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	mean := sum / float64(len(values))
	median := medianSorted(values)
	stddev := 0.0
	if len(values) > 1 {
		sumSquares := 0.0
		for _, value := range values {
			delta := value - mean
			sumSquares += delta * delta
		}
		stddev = math.Sqrt(sumSquares / float64(len(values)-1))
	}
	deviations := make([]float64, len(values))
	for i, value := range values {
		deviations[i] = math.Abs(value - median)
	}
	sort.Float64s(deviations)
	medianAbsDev := medianSorted(deviations)
	relStdDev := 0.0
	if mean != 0 {
		relStdDev = stddev / math.Abs(mean)
	}
	relMedianAbsDev := 0.0
	if median != 0 {
		relMedianAbsDev = medianAbsDev / math.Abs(median)
	}
	return metricSummary{
		Samples:         len(values),
		Median:          median,
		Mean:            mean,
		Min:             values[0],
		Max:             values[len(values)-1],
		StdDev:          stddev,
		RelStdDev:       relStdDev,
		MedianAbsDev:    medianAbsDev,
		RelMedianAbsDev: relMedianAbsDev,
	}
}

func medianSorted(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	}
	return median
}

func buildLanguageReports(summaries []benchmarkSummary, resources []runResource) []languageReport {
	byKey := make(map[key]benchmarkSummary, len(summaries))
	languages := map[string]bool{}
	for _, s := range summaries {
		byKey[key{suite: s.Suite, language: s.Language, backend: s.Backend}] = s
		languages[s.Language] = true
	}
	resourcesByLanguage := make(map[string][]runResource)
	for _, resource := range resources {
		if resource.Language == "" {
			continue
		}
		resourcesByLanguage[resource.Language] = append(resourcesByLanguage[resource.Language], resource)
		languages[resource.Language] = true
	}
	names := make([]string, 0, len(languages))
	for name := range languages {
		names = append(names, name)
	}
	sort.Strings(names)

	reports := make([]languageReport, 0, len(names))
	for _, name := range names {
		lr := languageReport{
			Language:       name,
			GoSuiteMetrics: map[string]map[string]float64{},
			CSuiteMetrics:  map[string]map[string]float64{},
		}
		lr.FullGoNanos = medianMetric(byKey, "Full", name, "gotreesitter", "ns/op")
		lr.FullCNanos = medianMetric(byKey, "Full", name, "tree-sitter-c", "ns/op")
		lr.EditGoNanos = medianMetric(byKey, "IncrementalSingleByteEdit", name, "gotreesitter", "ns/op")
		lr.EditCNanos = medianMetric(byKey, "IncrementalSingleByteEdit", name, "tree-sitter-c", "ns/op")
		lr.NoEditGoNanos = medianMetric(byKey, "IncrementalNoEdit", name, "gotreesitter", "ns/op")
		lr.NoEditCNanos = medianMetric(byKey, "IncrementalNoEdit", name, "tree-sitter-c", "ns/op")
		lr.FullRatio = ratio(lr.FullGoNanos, lr.FullCNanos)
		lr.EditRatio = ratio(lr.EditGoNanos, lr.EditCNanos)
		lr.NoEditRatio = ratio(lr.NoEditGoNanos, lr.NoEditCNanos)
		lr.WorstRatio = maxFloat(lr.FullRatio, lr.EditRatio, lr.NoEditRatio)
		lr.MinNSOpSamples, lr.MaxNSOpRelStdDev, lr.MaxNSOpRelMedianAbsDev, lr.QualityNotes = languageQuality(byKey, name)
		for _, resource := range resourcesByLanguage[name] {
			if resource.MaxRSSKB > lr.MaxRSSKB {
				lr.MaxRSSKB = resource.MaxRSSKB
			}
			if resource.OOMKilled != nil && *resource.OOMKilled {
				lr.OOMKilled = true
			}
			if resource.AllowMismatch != nil && *resource.AllowMismatch {
				lr.QualityNotes = appendUnique(lr.QualityNotes, "parity precheck disabled")
			}
			if resource.SkipMismatch != nil && *resource.SkipMismatch {
				lr.QualityNotes = appendUnique(lr.QualityNotes, "parity mismatch filtering enabled")
			}
			if resource.ExitCode != nil && *resource.ExitCode != 0 {
				lr.QualityNotes = appendUnique(lr.QualityNotes, fmt.Sprintf("container exit code %d", *resource.ExitCode))
			}
		}
		if lr.OOMKilled {
			lr.QualityNotes = appendUnique(lr.QualityNotes, "container OOM-killed")
		}
		for _, suite := range []string{"Full", "IncrementalSingleByteEdit", "IncrementalNoEdit"} {
			if metrics := medianMetrics(byKey, suite, name, "gotreesitter"); len(metrics) > 0 {
				lr.GoSuiteMetrics[suite] = metrics
			}
			if metrics := medianMetrics(byKey, suite, name, "tree-sitter-c"); len(metrics) > 0 {
				lr.CSuiteMetrics[suite] = metrics
			}
		}
		lr.TopAttribution = topAttribution(lr.GoSuiteMetrics, 6)
		reports = append(reports, lr)
	}
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].WorstRatio != reports[j].WorstRatio {
			return reports[i].WorstRatio > reports[j].WorstRatio
		}
		return reports[i].Language < reports[j].Language
	})
	return reports
}

func languageQuality(byKey map[key]benchmarkSummary, language string) (minSamples int, maxRelStdDev, maxRelMedianAbsDev float64, notes []string) {
	for _, suite := range []string{"Full", "IncrementalSingleByteEdit", "IncrementalNoEdit"} {
		for _, backend := range []string{"gotreesitter", "tree-sitter-c"} {
			summary, ok := byKey[key{suite: suite, language: language, backend: backend}]
			if !ok {
				notes = append(notes, fmt.Sprintf("missing %s/%s", suite, backend))
				continue
			}
			nsOp, ok := summary.Metrics["ns/op"]
			if !ok {
				notes = append(notes, fmt.Sprintf("missing %s/%s ns/op", suite, backend))
				continue
			}
			if minSamples == 0 || nsOp.Samples < minSamples {
				minSamples = nsOp.Samples
			}
			if nsOp.RelStdDev > maxRelStdDev {
				maxRelStdDev = nsOp.RelStdDev
			}
			if nsOp.RelMedianAbsDev > maxRelMedianAbsDev {
				maxRelMedianAbsDev = nsOp.RelMedianAbsDev
			}
		}
	}
	if minSamples > 0 && minSamples < 3 {
		notes = append(notes, fmt.Sprintf("low sample count n=%d", minSamples))
	}
	if maxRelStdDev > 0.05 {
		notes = append(notes, fmt.Sprintf("ns/op rel stddev %.1f%%", maxRelStdDev*100))
	}
	if maxRelMedianAbsDev > 0.03 {
		notes = append(notes, fmt.Sprintf("ns/op rel MAD %.1f%%", maxRelMedianAbsDev*100))
	}
	return minSamples, maxRelStdDev, maxRelMedianAbsDev, notes
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func medianMetric(byKey map[key]benchmarkSummary, suite, language, backend, metric string) float64 {
	s, ok := byKey[key{suite: suite, language: language, backend: backend}]
	if !ok {
		return 0
	}
	m, ok := s.Metrics[metric]
	if !ok {
		return 0
	}
	return m.Median
}

func medianMetrics(byKey map[key]benchmarkSummary, suite, language, backend string) map[string]float64 {
	s, ok := byKey[key{suite: suite, language: language, backend: backend}]
	if !ok {
		return nil
	}
	out := make(map[string]float64, len(s.Metrics))
	for name, m := range s.Metrics {
		out[name] = m.Median
	}
	return out
}

func ratio(goNanos, cNanos float64) float64 {
	if goNanos <= 0 || cNanos <= 0 {
		return 0
	}
	return goNanos / cNanos
}

func maxFloat(values ...float64) float64 {
	maxValue := 0.0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func topAttribution(suiteMetrics map[string]map[string]float64, limit int) []attributionBucket {
	names := []string{
		"parse_wall_ns/op",
		"reparse_ns/op",
		"reuse_ns/op",
		"unattributed_ns/op",
		"parser_loop_ns/op",
		"parser_accounted_ns/op",
		"parser_unattributed_ns/op",
		"token_next_ns/op",
		"action_dispatch_ns/op",
		"action_apply_ns/op",
		"action_lookup_ns/op",
		"glr_merge_ns/op",
		"glr_cull_ns/op",
		"result_accounted_ns/op",
		"result_select_ns/op",
		"result_tree_build_ns/op",
		"result_finalize_root_ns/op",
		"result_compatibility_ns/op",
		"result_parent_link_ns/op",
		"normalization_ns/op",
	}
	var buckets []attributionBucket
	for suite, metrics := range suiteMetrics {
		for _, name := range names {
			value := metrics[name]
			if value <= 0 {
				continue
			}
			buckets = append(buckets, attributionBucket{Suite: suite, Name: name, Value: value})
		}
	}
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].Value != buckets[j].Value {
			return buckets[i].Value > buckets[j].Value
		}
		if buckets[i].Suite != buckets[j].Suite {
			return buckets[i].Suite < buckets[j].Suite
		}
		return buckets[i].Name < buckets[j].Name
	})
	if len(buckets) > limit {
		buckets = buckets[:limit]
	}
	return buckets
}

func writeJSON(path string, r report) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeMarkdown(path string, r report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Real Corpus Bench Report\n\n")
	fmt.Fprintf(&b, "Generated: `%s`\n\n", r.GeneratedAt)
	fmt.Fprintf(&b, "Samples: `%d`\n\n", r.Samples)
	fmt.Fprintf(&b, "| Language | Full Go | Full C | Full xC | Edit Go | Edit C | Edit xC | No-edit Go | No-edit C | No-edit xC | Max RSS | Quality | Top attribution |\n")
	fmt.Fprintf(&b, "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|\n")
	for _, lr := range r.Languages {
		fmt.Fprintf(
			&b,
			"| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			lr.Language,
			formatNanos(lr.FullGoNanos),
			formatNanos(lr.FullCNanos),
			formatRatio(lr.FullRatio),
			formatNanos(lr.EditGoNanos),
			formatNanos(lr.EditCNanos),
			formatRatio(lr.EditRatio),
			formatNanos(lr.NoEditGoNanos),
			formatNanos(lr.NoEditCNanos),
			formatRatio(lr.NoEditRatio),
			formatRSSKB(lr.MaxRSSKB),
			formatQuality(lr),
			formatAttribution(lr.TopAttribution),
		)
	}
	fmt.Fprintf(&b, "\n## Inputs\n\n")
	for _, input := range r.Inputs {
		fmt.Fprintf(&b, "- `%s`\n", input)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func formatNanos(v float64) string {
	if v <= 0 {
		return ""
	}
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.3fs", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.3fms", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.3fus", v/1e3)
	default:
		return fmt.Sprintf("%.0fns", v)
	}
}

func formatRatio(v float64) string {
	if v <= 0 {
		return ""
	}
	return fmt.Sprintf("%.2fx", v)
}

func formatRSSKB(kb int64) string {
	if kb <= 0 {
		return ""
	}
	const mib = 1024
	const gib = 1024 * mib
	if kb >= gib {
		return fmt.Sprintf("%.2fGiB", float64(kb)/float64(gib))
	}
	if kb >= mib {
		return fmt.Sprintf("%.1fMiB", float64(kb)/float64(mib))
	}
	return fmt.Sprintf("%dKiB", kb)
}

func formatQuality(lr languageReport) string {
	parts := make([]string, 0, 3+len(lr.QualityNotes))
	if lr.MinNSOpSamples > 0 {
		parts = append(parts, fmt.Sprintf("n>=%d", lr.MinNSOpSamples))
	}
	if lr.MaxNSOpRelStdDev > 0 {
		parts = append(parts, fmt.Sprintf("max RSD %.1f%%", lr.MaxNSOpRelStdDev*100))
	}
	if lr.MaxNSOpRelMedianAbsDev > 0 {
		parts = append(parts, fmt.Sprintf("max MAD %.1f%%", lr.MaxNSOpRelMedianAbsDev*100))
	}
	parts = append(parts, lr.QualityNotes...)
	return strings.Join(parts, "<br>")
}

func formatAttribution(buckets []attributionBucket) string {
	if len(buckets) == 0 {
		return ""
	}
	parts := make([]string, 0, len(buckets))
	for _, bucket := range buckets {
		parts = append(parts, fmt.Sprintf("%s/%s=%s", bucket.Suite, bucket.Name, formatNanos(bucket.Value)))
	}
	return strings.Join(parts, "<br>")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
