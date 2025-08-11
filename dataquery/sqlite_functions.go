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

func init() {
	sqlitecore.RegisterScalarFunction("k8s_cpu_to_number", 1, func(ctx *sqlitecore.FunctionContext, args []driver.Value) (driver.Value, error) {
		return k8sCPUToNumber(args[0].(string)), nil
	})
}
