package dataquery

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	sqlitecore "modernc.org/sqlite"
)

// k8sMillicores converts Kubernetes CPU units to decimal numbers using resource.Quantity
// Examples: "500m" -> 500, "1" -> 1000, "2000m" -> 2000
func k8sMillicores(cpuStr string) float64 {
	if cpuStr == "" {
		return 0
	}

	// Try to parse as a Kubernetes resource quantity
	quantity, err := resource.ParseQuantity(cpuStr)
	if err != nil {
		// Fallback: try parsing as plain number for backwards compatibility
		if cpu, parseErr := strconv.ParseFloat(cpuStr, 64); parseErr == nil {
			return cpu
		}
		return 0
	}

	return float64(quantity.MilliValue())
}

// memoryToBytes converts memory strings with units to bytes using resource.Quantity
// Examples: "500" -> 500, "500KB" -> 500000, "500MB" -> 500000000, "5Gi" -> 5368709120
func memoryToBytes(memoryStr string) int64 {
	if memoryStr == "" {
		return 0
	}

	// Remove all spaces for flexible parsing
	memoryStr = strings.ReplaceAll(memoryStr, " ", "")

	// Handle plain numbers (already in bytes) - try this first for backwards compatibility
	if val, err := strconv.ParseInt(memoryStr, 10, 64); err == nil {
		return val
	}

	// Try to parse as a Kubernetes resource quantity first (supports K, M, G, T, Ki, Mi, Gi, Ti)
	quantity, err := resource.ParseQuantity(memoryStr)
	if err == nil {
		value := quantity.Value()
		return value
	}

	// Fallback: handle other units that Kubernetes doesn't support (KB, MB, GB, TB, etc.)
	memoryStr = strings.ToUpper(memoryStr)

	// Handle various unit suffixes
	var multiplier int64
	var numStr string

	switch {
	case strings.HasSuffix(memoryStr, "KB"):
		multiplier = 1000
		numStr = strings.TrimSuffix(memoryStr, "KB")
	case strings.HasSuffix(memoryStr, "MB"):
		multiplier = 1000 * 1000
		numStr = strings.TrimSuffix(memoryStr, "MB")
	case strings.HasSuffix(memoryStr, "GB"):
		multiplier = 1000 * 1000 * 1000
		numStr = strings.TrimSuffix(memoryStr, "GB")
	case strings.HasSuffix(memoryStr, "TB"):
		multiplier = 1000 * 1000 * 1000 * 1000
		numStr = strings.TrimSuffix(memoryStr, "TB")
	case strings.HasSuffix(memoryStr, "KIB"):
		multiplier = 1024
		numStr = strings.TrimSuffix(memoryStr, "KIB")
	case strings.HasSuffix(memoryStr, "MIB"):
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(memoryStr, "MIB")
	case strings.HasSuffix(memoryStr, "GIB"):
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(memoryStr, "GIB")
	case strings.HasSuffix(memoryStr, "TIB"):
		multiplier = 1024 * 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(memoryStr, "TIB")
	case strings.HasSuffix(memoryStr, "K"):
		multiplier = 1000
		numStr = strings.TrimSuffix(memoryStr, "K")
	case strings.HasSuffix(memoryStr, "M"):
		multiplier = 1000 * 1000
		numStr = strings.TrimSuffix(memoryStr, "M")
	case strings.HasSuffix(memoryStr, "G"):
		multiplier = 1000 * 1000 * 1000
		numStr = strings.TrimSuffix(memoryStr, "G")
	case strings.HasSuffix(memoryStr, "T"):
		multiplier = 1000 * 1000 * 1000 * 1000
		numStr = strings.TrimSuffix(memoryStr, "T")
	default:
		return 0
	}

	numStr = strings.TrimSpace(numStr)
	if val, err := strconv.ParseInt(numStr, 10, 64); err == nil {
		return val * multiplier
	}

	return 0
}

func init() {
	_ = sqlitecore.RegisterScalarFunction("to_millicores", 1, func(ctx *sqlitecore.FunctionContext, args []driver.Value) (driver.Value, error) {
		if args[0] == nil {
			return 0.0, nil
		}

		var str string
		switch v := args[0].(type) {
		case string:
			str = v
		case int64:
			str = strconv.FormatInt(v, 10)
		case float64:
			str = fmt.Sprintf("%g", v)
		default:
			str = fmt.Sprintf("%v", v)
		}

		return k8sMillicores(str), nil
	})

	_ = sqlitecore.RegisterScalarFunction("to_bytes", 1, func(ctx *sqlitecore.FunctionContext, args []driver.Value) (driver.Value, error) {
		if args[0] == nil {
			return int64(0), nil
		}

		var str string
		switch v := args[0].(type) {
		case string:
			str = v
		case int64:
			str = strconv.FormatInt(v, 10)
		case float64:
			str = fmt.Sprintf("%g", v)
		default:
			str = fmt.Sprintf("%v", v)
		}

		return memoryToBytes(str), nil
	})
}
