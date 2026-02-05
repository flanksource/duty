package tokenizer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

type Replacement struct {
	Value string
	Regex *regexp.Regexp
}

type Replacements []Replacement

var tokenizer Replacements

func NewReplacements(pairs ...string) Replacements {
	var r Replacements
	for i := 0; i < len(pairs)-1; i = i + 2 {
		r = append(r, Replacement{
			Value: pairs[i],
			Regex: regexp.MustCompile(pairs[i+1]),
		})
	}
	return r
}

func (replacements Replacements) Tokenize(data any) string {
	switch v := data.(type) {

	case int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64:
		return "0"
	case time.Duration:
		return "DURATION"
	case time.Time:
		return "TIMESTAMP"
	case string:
		out := v
		for _, r := range replacements {
			out = r.Regex.ReplaceAllString(out, r.Value)
			if out == r.Value {
				break
			}
		}
		return out
	}

	return fmt.Sprintf("%v", data)
}

func Tokenize(data any) string {
	return tokenizer.Tokenize(data)
}

func TokenizedHash(data any) string {
	h := sha256.New()
	s := Tokenize(data)
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func init() {
	tokenizer = NewReplacements(
		"UUID", `\b[0-9a-f]{8}\b-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-\b[0-9a-f]{12}\b`,
		"TIMESTAMP", `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})`,
		"DURATION", `\s+\d+(.\d+){0,1}(ms|s|h|d|m)`,
		"SHA256", `[a-z0-9]{64}`,
		"NUMBER", `^\d+$`,
		"HEX16", `[0-9a-f]{16}`, // matches a 16 character long hex string
	)
}
