package environment

import (
	"os"
	"strconv"
	"time"
)

func StringOrDefault(key, defaultValue string) string {
	envVal := os.Getenv(key)
	if envVal == "" {
		return defaultValue
	}
	return envVal
}

func BoolOrDefault(key string, defaultValue bool) bool {
	envVal := os.Getenv(key)
	if envVal == "" {
		return defaultValue
	}
	r, err := strconv.ParseBool(envVal)
	if err != nil {
		return defaultValue
	}
	return r
}

func IntOrDefault(key string, defaultValue int) int {
	envVal := os.Getenv(key)
	if envVal == "" {
		return defaultValue
	}
	r, err := strconv.Atoi(envVal)
	if err != nil {
		return defaultValue
	}
	return r
}

// DurationOrDefault returns a time.Duration that is determined by using the time.ParseDuration function.
func DurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	envVal := os.Getenv(key)
	if envVal == "" {
		return defaultValue
	}
	r, err := time.ParseDuration(envVal)
	if err != nil {
		return defaultValue
	}
	return r
}
