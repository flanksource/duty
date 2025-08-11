package dataquery

import (
	"database/sql/driver"
	"strconv"
	"strings"

	sqlitecore "modernc.org/sqlite"
)

// k8sCPUToNumber converts Kubernetes CPU units to decimal numbers
// Examples: "500m" -> 0.5, "1" -> 1.0, "2000m" -> 2.0
func k8sCPUToNumber(cpuStr string) float64 {
	if cpuStr == "" {
		return 0
	}

	if strings.HasSuffix(cpuStr, "m") {
		milliStr := strings.TrimSuffix(cpuStr, "m")
		if milli, err := strconv.ParseFloat(milliStr, 64); err == nil {
			return milli / 1000.0
		}
		return 0
	}

	if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
		return cpu
	}

	return 0
}

// memoryToBytes converts memory strings with units to bytes
// Examples: "500" -> 500, "500KB" -> 500000, "500MB" -> 500000000
func memoryToBytes(memoryStr string) int64 {
	if memoryStr == "" {
		return 0
	}

	// Handle plain numbers (already in bytes)
	if val, err := strconv.ParseInt(memoryStr, 10, 64); err == nil {
		return val
	}

	memoryStr = strings.ToUpper(memoryStr)

	// Handle various unit suffixes
	var multiplier int64 = 1
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

	if val, err := strconv.ParseInt(numStr, 10, 64); err == nil {
		return val * multiplier
	}

	return 0
}

func init() {
	sqlitecore.RegisterScalarFunction("k8s_cpu_to_number", 1, func(ctx *sqlitecore.FunctionContext, args []driver.Value) (driver.Value, error) {
		return k8sCPUToNumber(args[0].(string)), nil
	})

	sqlitecore.RegisterScalarFunction("memory_to_bytes", 1, func(ctx *sqlitecore.FunctionContext, args []driver.Value) (driver.Value, error) {
		return memoryToBytes(args[0].(string)), nil
	})
}
