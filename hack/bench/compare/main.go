package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type result struct {
	name  string
	base  int64
	head  int64
	delta float64
	trend string
	bar   string
}

func parseBench(path string) (map[string]int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	results := make(map[string]int64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}
		fields := strings.Fields(line)
		nsIdx := -1
		for i, field := range fields {
			if field == "ns/op" {
				nsIdx = i
				break
			}
		}
		if nsIdx <= 0 {
			continue
		}
		value, err := strconv.ParseInt(fields[nsIdx-1], 10, 64)
		if err != nil {
			continue
		}
		results[fields[0]] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func formatNS(value int64) string {
	switch {
	case value >= 1_000_000_000:
		return fmt.Sprintf("%.2fs", float64(value)/1_000_000_000)
	case value >= 1_000_000:
		return fmt.Sprintf("%.2fms", float64(value)/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.2fus", float64(value)/1_000)
	default:
		return fmt.Sprintf("%dns", value)
	}
}

func classify(delta float64, threshold float64) string {
	if delta <= -threshold {
		return "improved"
	}
	if delta >= threshold {
		return "regressed"
	}
	return "neutral"
}

func trendBar(delta float64) string {
	if delta == 0 {
		return "="
	}
	magnitude := int(math.Abs(delta) / 5.0)
	if magnitude < 1 {
		magnitude = 1
	}
	if magnitude > 10 {
		magnitude = 10
	}
	if delta > 0 {
		return strings.Repeat("+", magnitude)
	}
	return strings.Repeat("-", magnitude)
}

func main() {
	basePath := flag.String("base", "", "base benchmark output")
	headPath := flag.String("head", "", "head benchmark output")
	outputPath := flag.String("output", "", "markdown output path")
	topN := flag.Int("top", 10, "top regressions/improvements to show")
	threshold := flag.Float64("threshold", 5.0, "percent threshold")
	baseRef := flag.String("base-ref", "", "label for base")
	headRef := flag.String("head-ref", "", "label for head")
	benchstatPath := flag.String("benchstat", "", "benchstat output path")
	flag.Parse()

	if *basePath == "" || *headPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "base, head, and output are required")
		os.Exit(2)
	}

	base, err := parseBench(*basePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read base: %v\n", err)
		os.Exit(1)
	}
	head, err := parseBench(*headPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read head: %v\n", err)
		os.Exit(1)
	}

	baseOnly := make([]string, 0)
	headOnly := make([]string, 0)
	common := make([]string, 0)

	for name := range base {
		if _, ok := head[name]; ok {
			common = append(common, name)
		} else {
			baseOnly = append(baseOnly, name)
		}
	}
	for name := range head {
		if _, ok := base[name]; !ok {
			headOnly = append(headOnly, name)
		}
	}

	sort.Strings(baseOnly)
	sort.Strings(headOnly)
	sort.Strings(common)

	rows := make([]result, 0, len(common))
	for _, name := range common {
		baseNS := base[name]
		headNS := head[name]
		if baseNS == 0 {
			continue
		}
		delta := (float64(headNS)/float64(baseNS) - 1.0) * 100.0
		rows = append(rows, result{
			name:  name,
			base:  baseNS,
			head:  headNS,
			delta: delta,
			trend: classify(delta, *threshold),
			bar:   trendBar(delta),
		})
	}

	regressions := make([]result, 0)
	improvements := make([]result, 0)
	neutral := make([]result, 0)
	for _, row := range rows {
		switch row.trend {
		case "regressed":
			regressions = append(regressions, row)
		case "improved":
			improvements = append(improvements, row)
		default:
			neutral = append(neutral, row)
		}
	}

	sort.Slice(regressions, func(i, j int) bool { return regressions[i].delta > regressions[j].delta })
	sort.Slice(improvements, func(i, j int) bool { return improvements[i].delta < improvements[j].delta })

	geomeanDelta := math.NaN()
	if len(rows) > 0 {
		logSum := 0.0
		count := 0
		for _, row := range rows {
			if row.base == 0 {
				continue
			}
			logSum += math.Log(float64(row.head) / float64(row.base))
			count++
		}
		if count > 0 {
			geomeanDelta = (math.Exp(logSum/float64(count)) - 1.0) * 100.0
		}
	}

	baseLabel := *baseRef
	if baseLabel == "" {
		baseLabel = *basePath
	}
	headLabel := *headRef
	if headLabel == "" {
		headLabel = *headPath
	}

	var builder strings.Builder
	builder.WriteString("<!-- benchmark-report -->\n")
	builder.WriteString("# Benchmark comparison\n\n")
	builder.WriteString(fmt.Sprintf("Base: `%s`\n", baseLabel))
	builder.WriteString(fmt.Sprintf("Head: `%s`\n\n", headLabel))

	builder.WriteString("## Summary\n")
	builder.WriteString(fmt.Sprintf("- total: %d\n", len(rows)))
	builder.WriteString(fmt.Sprintf("- improved (<= -%.1f%%): %d\n", *threshold, len(improvements)))
	builder.WriteString(fmt.Sprintf("- regressed (>= +%.1f%%): %d\n", *threshold, len(regressions)))
	builder.WriteString(fmt.Sprintf("- neutral: %d\n", len(neutral)))
	if !math.IsNaN(geomeanDelta) {
		direction := "flat"
		if geomeanDelta < 0 {
			direction = "faster"
		} else if geomeanDelta > 0 {
			direction = "slower"
		}
		builder.WriteString(fmt.Sprintf("- geomean change: %+0.1f%% (%s)\n", geomeanDelta, direction))
	}

	if len(baseOnly) > 0 {
		builder.WriteString("\n## Missing in head\n")
		limit := *topN
		if limit > len(baseOnly) {
			limit = len(baseOnly)
		}
		for _, name := range baseOnly[:limit] {
			builder.WriteString(fmt.Sprintf("- %s\n", name))
		}
		if len(baseOnly) > limit {
			builder.WriteString(fmt.Sprintf("- ...and %d more\n", len(baseOnly)-limit))
		}
	}

	if len(headOnly) > 0 {
		builder.WriteString("\n## New in head\n")
		limit := *topN
		if limit > len(headOnly) {
			limit = len(headOnly)
		}
		for _, name := range headOnly[:limit] {
			builder.WriteString(fmt.Sprintf("- %s\n", name))
		}
		if len(headOnly) > limit {
			builder.WriteString(fmt.Sprintf("- ...and %d more\n", len(headOnly)-limit))
		}
	}

	writeTable := func(title string, items []result) {
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("### %s\n\n", title))
		if len(items) == 0 {
			builder.WriteString("None\n")
			return
		}
		builder.WriteString("| Benchmark | Base | Head | Delta | Trend |\n")
		builder.WriteString("| --- | --- | --- | --- | --- |\n")
		limit := *topN
		if limit > len(items) {
			limit = len(items)
		}
		for _, item := range items[:limit] {
			builder.WriteString(fmt.Sprintf(
				"| %s | %s | %s | %+0.1f%% | %s %s |\n",
				item.name,
				formatNS(item.base),
				formatNS(item.head),
				item.delta,
				item.trend,
				item.bar,
			))
		}
	}

	writeTable("Top regressions", regressions)
	writeTable("Top improvements", improvements)

	builder.WriteString("\n## Notes\n")
	builder.WriteString("- Delta is based on ns/op (lower is better).\n")
	builder.WriteString("- Trend bar uses +/- to show magnitude (each char ~5%).\n")

	if *benchstatPath != "" {
		content, err := os.ReadFile(*benchstatPath)
		if err == nil {
			builder.WriteString("\n## Benchstat\n\n<details>\n<summary>benchstat output</summary>\n\n```text\n")
			builder.WriteString(strings.TrimSpace(string(content)))
			builder.WriteString("\n```\n</details>\n")
		}
	}

	if err := os.WriteFile(*outputPath, []byte(builder.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write report: %v\n", err)
		os.Exit(1)
	}
}
