package tracing

import "unicode/utf8"

// DefaultMaxSQLStatementLength caps SQL statement span attributes at 8 KiB.
const DefaultMaxSQLStatementLength = 8 * 1024

const truncatedSQLStatementSuffix = " ... [truncated]"

// SQLStatementTruncator returns a query formatter that caps SQL statement size.
func SQLStatementTruncator(maxLength int) func(statement string) string {
	return func(statement string) string {
		return truncateSQLStatement(statement, maxLength)
	}
}

func truncateSQLStatement(statement string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}
	if len(statement) <= maxLength {
		return statement
	}

	if maxLength <= len(truncatedSQLStatementSuffix) {
		return truncateUTF8(statement, maxLength)
	}

	return truncateUTF8(statement, maxLength-len(truncatedSQLStatementSuffix)) + truncatedSQLStatementSuffix
}

func truncateUTF8(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}

	end := maxBytes
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end]
}
