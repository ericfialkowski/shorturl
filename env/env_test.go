package env

import (
	"os"
	"testing"
	"time"
)

func TestStringOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue string
		expected     string
		setEnv       bool
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_STRING_UNSET",
			defaultValue: "default",
			expected:     "default",
			setEnv:       false,
		},
		{
			name:         "returns env value when set",
			key:          "TEST_STRING_SET",
			envValue:     "custom",
			defaultValue: "default",
			expected:     "custom",
			setEnv:       true,
		},
		{
			name:         "returns default when env is empty string",
			key:          "TEST_STRING_EMPTY",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
			setEnv:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := StringOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("StringOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBoolOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue bool
		expected     bool
		setEnv       bool
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_BOOL_UNSET",
			defaultValue: true,
			expected:     true,
			setEnv:       false,
		},
		{
			name:         "returns true when env is 'true'",
			key:          "TEST_BOOL_TRUE",
			envValue:     "true",
			defaultValue: false,
			expected:     true,
			setEnv:       true,
		},
		{
			name:         "returns false when env is 'false'",
			key:          "TEST_BOOL_FALSE",
			envValue:     "false",
			defaultValue: true,
			expected:     false,
			setEnv:       true,
		},
		{
			name:         "returns true when env is '1'",
			key:          "TEST_BOOL_ONE",
			envValue:     "1",
			defaultValue: false,
			expected:     true,
			setEnv:       true,
		},
		{
			name:         "returns false when env is '0'",
			key:          "TEST_BOOL_ZERO",
			envValue:     "0",
			defaultValue: true,
			expected:     false,
			setEnv:       true,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_BOOL_INVALID",
			envValue:     "notabool",
			defaultValue: true,
			expected:     true,
			setEnv:       true,
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_BOOL_EMPTY",
			envValue:     "",
			defaultValue: true,
			expected:     true,
			setEnv:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := BoolOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("BoolOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue int
		expected     int
		setEnv       bool
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_INT_UNSET",
			defaultValue: 42,
			expected:     42,
			setEnv:       false,
		},
		{
			name:         "returns env value when set",
			key:          "TEST_INT_SET",
			envValue:     "100",
			defaultValue: 42,
			expected:     100,
			setEnv:       true,
		},
		{
			name:         "returns negative value",
			key:          "TEST_INT_NEGATIVE",
			envValue:     "-50",
			defaultValue: 42,
			expected:     -50,
			setEnv:       true,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_INT_INVALID",
			envValue:     "notanint",
			defaultValue: 42,
			expected:     42,
			setEnv:       true,
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_INT_EMPTY",
			envValue:     "",
			defaultValue: 42,
			expected:     42,
			setEnv:       true,
		},
		{
			name:         "returns default when env is float",
			key:          "TEST_INT_FLOAT",
			envValue:     "3.14",
			defaultValue: 42,
			expected:     42,
			setEnv:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := IntOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("IntOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDurationOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
		setEnv       bool
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_DURATION_UNSET",
			defaultValue: 5 * time.Second,
			expected:     5 * time.Second,
			setEnv:       false,
		},
		{
			name:         "parses seconds",
			key:          "TEST_DURATION_SECONDS",
			envValue:     "10s",
			defaultValue: 5 * time.Second,
			expected:     10 * time.Second,
			setEnv:       true,
		},
		{
			name:         "parses minutes",
			key:          "TEST_DURATION_MINUTES",
			envValue:     "2m",
			defaultValue: 5 * time.Second,
			expected:     2 * time.Minute,
			setEnv:       true,
		},
		{
			name:         "parses milliseconds",
			key:          "TEST_DURATION_MS",
			envValue:     "500ms",
			defaultValue: 5 * time.Second,
			expected:     500 * time.Millisecond,
			setEnv:       true,
		},
		{
			name:         "parses complex duration",
			key:          "TEST_DURATION_COMPLEX",
			envValue:     "1h30m",
			defaultValue: 5 * time.Second,
			expected:     90 * time.Minute,
			setEnv:       true,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_DURATION_INVALID",
			envValue:     "notaduration",
			defaultValue: 5 * time.Second,
			expected:     5 * time.Second,
			setEnv:       true,
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_DURATION_EMPTY",
			envValue:     "",
			defaultValue: 5 * time.Second,
			expected:     5 * time.Second,
			setEnv:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := DurationOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("DurationOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}
