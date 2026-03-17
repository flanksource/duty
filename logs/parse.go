package logs

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/flanksource/commons/utils"
)

var klogPattern = regexp.MustCompile(`^([IWEF])(\d{4})\s+(\d{2}:\d{2}:\d{2}\.\d+)\s+(\d+)\s+(\S+:\d+)]\s*(.*)$`)

var klogSeverities = map[byte]string{
	'I': "info",
	'W': "warning",
	'E': "error",
	'F': "fatal",
}

// syslog: RFC3164-style <priority>timestamp hostname app[pid]: message
// Also handles common variants without priority or pid.
var syslogPattern = regexp.MustCompile(
	`^(?:<(\d+)>)?` + // optional priority
		`(\w{3}\s+\d+\s+\d{2}:\d{2}:\d{2})\s+` + // timestamp
		`(\S+)\s+` + // hostname
		`(\S+?)(?:\[(\d+)\])?:\s+` + // app[pid]:
		`(.*)$`) // message

var syslogSeverities = []string{
	"emergency", "alert", "critical", "error",
	"warning", "notice", "info", "debug",
}

func ParseMessage(line *LogLine, format string) {
	switch format {
	case "klogfmt":
		ParseKlogfmt(line)
	case "logfmt":
		ParseLogfmt(line)
	case "json":
		ParseJSON(line)
	case "syslog":
		ParseSyslog(line)
	case "autodetect", "":
		ParseAutodetect(line)
	}
}

func DetectFormat(msg string) string {
	if len(msg) == 0 {
		return ""
	}
	if msg[0] == '{' {
		return "json"
	}
	if klogPattern.MatchString(msg) {
		return "klogfmt"
	}
	if msg[0] == '<' && syslogPattern.MatchString(msg) {
		return "syslog"
	}
	if pairs := parseKeyValuePairs(msg); len(pairs) >= 2 {
		return "logfmt"
	}
	return ""
}

func ParseAutodetect(line *LogLine) {
	if format := DetectFormat(line.Message); format != "" {
		ParseMessage(line, format)
	}
}

func ParseKlogfmt(line *LogLine) {
	matches := klogPattern.FindStringSubmatch(line.Message)
	if matches == nil {
		return
	}

	if sev, ok := klogSeverities[matches[1][0]]; ok {
		line.Severity = sev
	}
	line.Source = matches[5]

	remaining := matches[6]
	msg, kvPairs := parseKlogMessage(remaining)
	if msg != "" {
		line.Message = msg
	}

	if len(kvPairs) > 0 {
		if line.Labels == nil {
			line.Labels = make(map[string]string)
		}
		for k, v := range kvPairs {
			line.Labels[k] = v
		}
	}
}

func parseKlogMessage(s string) (string, map[string]string) {
	kvPairs := make(map[string]string)
	var msg string

	if strings.HasPrefix(s, "\"") {
		end := strings.Index(s[1:], "\"")
		if end >= 0 {
			msg = s[1 : end+1]
			s = strings.TrimSpace(s[end+2:])
		}
	}

	for _, pair := range parseKeyValuePairs(s) {
		kvPairs[pair[0]] = pair[1]
	}

	if msg == "" && len(kvPairs) == 0 {
		msg = s
	}

	return msg, kvPairs
}

func parseKeyValuePairs(s string) [][2]string {
	var pairs [][2]string
	for len(s) > 0 {
		s = strings.TrimSpace(s)
		if s == "" {
			break
		}

		eqIdx := strings.Index(s, "=")
		if eqIdx < 0 {
			break
		}

		key := s[:eqIdx]
		if strings.ContainsAny(key, " \t") {
			break
		}

		s = s[eqIdx+1:]

		var value string
		if strings.HasPrefix(s, "\"") {
			end := strings.Index(s[1:], "\"")
			if end >= 0 {
				value = s[1 : end+1]
				s = s[end+2:]
			} else {
				value = s[1:]
				s = ""
			}
		} else {
			spaceIdx := strings.IndexAny(s, " \t")
			if spaceIdx >= 0 {
				value = s[:spaceIdx]
				s = s[spaceIdx:]
			} else {
				value = s
				s = ""
			}
		}

		pairs = append(pairs, [2]string{key, value})
	}
	return pairs
}

func ParseLogfmt(line *LogLine) {
	pairs := parseKeyValuePairs(line.Message)
	if len(pairs) == 0 {
		return
	}

	if line.Labels == nil {
		line.Labels = make(map[string]string)
	}

	for _, pair := range pairs {
		key, value := pair[0], pair[1]
		switch key {
		case "msg", "message":
			line.Message = value
		case "level":
			line.Severity = value
		default:
			line.Labels[key] = value
		}
	}
}

func ParseJSON(line *LogLine) {
	var fields map[string]any
	if err := json.Unmarshal([]byte(line.Message), &fields); err != nil {
		return
	}

	if line.Labels == nil {
		line.Labels = make(map[string]string)
	}

	for key, val := range fields {
		str, _ := utils.Stringify(val)
		switch key {
		case "msg", "message":
			line.Message = str
		case "level", "severity":
			line.Severity = str
		case "source", "caller", "logger":
			line.Source = str
		case "host", "hostname":
			line.Host = str
		default:
			line.Labels[key] = str
		}
	}
}

func ParseSyslog(line *LogLine) {
	matches := syslogPattern.FindStringSubmatch(line.Message)
	if matches == nil {
		return
	}

	if line.Labels == nil {
		line.Labels = make(map[string]string)
	}

	if matches[1] != "" {
		pri := 0
		for _, c := range matches[1] {
			pri = pri*10 + int(c-'0')
		}
		severity := pri & 0x07
		if severity < len(syslogSeverities) {
			line.Severity = syslogSeverities[severity]
		}
	}

	line.Host = matches[3]
	line.Source = matches[4]
	if matches[5] != "" {
		line.Labels["pid"] = matches[5]
	}
	line.Message = matches[6]
}
