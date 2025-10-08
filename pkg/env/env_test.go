package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetString(t *testing.T) {
	t.Run("Return environment variable when exists", func(t *testing.T) {
		key := "TEST_STRING_VAR"
		expected := "test_value"

		err := os.Setenv(key, expected)
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetString(key, "default")

		assert.Equal(t, expected, result)
	})

	t.Run("Return default value when environment variable doesn't exist", func(t *testing.T) {
		key := "NON_EXISTENT_VAR"
		defaultValue := "default_value"

		// Ensure the environment variable doesn't exist
		os.Unsetenv(key)

		result := GetString(key, defaultValue)

		assert.Equal(t, defaultValue, result)
	})

	t.Run("Return empty string when environment variable is empty", func(t *testing.T) {
		key := "EMPTY_STRING_VAR"

		err := os.Setenv(key, "")
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetString(key, "default")

		assert.Equal(t, "", result)
	})

	t.Run("Return environment variable with spaces", func(t *testing.T) {
		key := "SPACES_VAR"
		expected := "  value with spaces  "

		err := os.Setenv(key, expected)
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetString(key, "default")

		assert.Equal(t, expected, result)
	})
}

func TestGetInt(t *testing.T) {
	t.Run("Return parsed integer when valid", func(t *testing.T) {
		key := "TEST_INT_VAR"
		expected := 42

		err := os.Setenv(key, "42")
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetInt(key, 0)

		assert.Equal(t, expected, result)
	})

	t.Run("Return default value when environment variable doesn't exist", func(t *testing.T) {
		key := "NON_EXISTENT_INT_VAR"
		defaultValue := 100

		// Ensure the environment variable doesn't exist
		os.Unsetenv(key)

		result := GetInt(key, defaultValue)

		assert.Equal(t, defaultValue, result)
	})

	t.Run("Return negative integer when valid", func(t *testing.T) {
		key := "NEGATIVE_INT_VAR"
		expected := -123

		err := os.Setenv(key, "-123")
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetInt(key, 0)

		assert.Equal(t, expected, result)
	})

	t.Run("Return zero when environment variable is zero", func(t *testing.T) {
		key := "ZERO_INT_VAR"

		err := os.Setenv(key, "0")
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetInt(key, 999)

		assert.Equal(t, 0, result)
	})

	t.Run("Panic when environment variable is not a valid integer", func(t *testing.T) {
		key := "INVALID_INT_VAR"

		err := os.Setenv(key, "not_an_integer")
		require.NoError(t, err)
		defer os.Unsetenv(key)

		assert.Panics(t, func() {
			GetInt(key, 0)
		})
	})

	t.Run("Panic when environment variable is empty string", func(t *testing.T) {
		key := "EMPTY_INT_VAR"

		err := os.Setenv(key, "")
		require.NoError(t, err)
		defer os.Unsetenv(key)

		assert.Panics(t, func() {
			GetInt(key, 0)
		})
	})

	t.Run("Panic when environment variable has spaces", func(t *testing.T) {
		key := "SPACES_INT_VAR"

		err := os.Setenv(key, " 42 ")
		require.NoError(t, err)
		defer os.Unsetenv(key)

		assert.Panics(t, func() {
			GetInt(key, 0)
		})
	})

	t.Run("Panic when environment variable is floating point", func(t *testing.T) {
		key := "FLOAT_INT_VAR"

		err := os.Setenv(key, "42.5")
		require.NoError(t, err)
		defer os.Unsetenv(key)

		assert.Panics(t, func() {
			GetInt(key, 0)
		})
	})
}

func TestGetBool(t *testing.T) {
	t.Run("Return true for valid true values", func(t *testing.T) {
		trueValues := []string{"true", "TRUE", "True", "1", "t", "T"}

		for _, value := range trueValues {
			t.Run("Value: "+value, func(t *testing.T) {
				key := "TEST_BOOL_VAR_TRUE"

				err := os.Setenv(key, value)
				require.NoError(t, err)
				defer os.Unsetenv(key)

				result := GetBool(key, false)

				assert.True(t, result)
			})
		}
	})

	t.Run("Return false for valid false values", func(t *testing.T) {
		falseValues := []string{"false", "FALSE", "False", "0", "f", "F"}

		for _, value := range falseValues {
			t.Run("Value: "+value, func(t *testing.T) {
				key := "TEST_BOOL_VAR_FALSE"

				err := os.Setenv(key, value)
				require.NoError(t, err)
				defer os.Unsetenv(key)

				result := GetBool(key, true)

				assert.False(t, result)
			})
		}
	})

	t.Run("Return default value when environment variable doesn't exist", func(t *testing.T) {
		key := "NON_EXISTENT_BOOL_VAR"
		defaultValue := true

		// Ensure the environment variable doesn't exist
		os.Unsetenv(key)

		result := GetBool(key, defaultValue)

		assert.Equal(t, defaultValue, result)
	})

	t.Run("Panic when environment variable is not a valid boolean", func(t *testing.T) {
		invalidValues := []string{"yes", "no", "on", "off", "invalid", "2", "-1", ""}

		for _, value := range invalidValues {
			t.Run("Invalid value: "+value, func(t *testing.T) {
				key := "INVALID_BOOL_VAR"

				err := os.Setenv(key, value)
				require.NoError(t, err)
				defer os.Unsetenv(key)

				assert.Panics(t, func() {
					GetBool(key, false)
				})
			})
		}
	})
}

// Integration tests
func TestEnvironmentVariableIntegration(t *testing.T) {
	t.Run("Multiple environment variables", func(t *testing.T) {
		// Set up multiple environment variables
		stringKey := "TEST_STRING"
		intKey := "TEST_INT"
		boolKey := "TEST_BOOL"

		err := os.Setenv(stringKey, "hello world")
		require.NoError(t, err)
		defer os.Unsetenv(stringKey)

		err = os.Setenv(intKey, "999")
		require.NoError(t, err)
		defer os.Unsetenv(intKey)

		err = os.Setenv(boolKey, "true")
		require.NoError(t, err)
		defer os.Unsetenv(boolKey)

		// Test all functions
		stringResult := GetString(stringKey, "default")
		intResult := GetInt(intKey, 0)
		boolResult := GetBool(boolKey, false)

		assert.Equal(t, "hello world", stringResult)
		assert.Equal(t, 999, intResult)
		assert.True(t, boolResult)
	})

	t.Run("Mix of existing and non-existing variables", func(t *testing.T) {
		existingKey := "EXISTING_VAR"
		nonExistingKey := "NON_EXISTING_VAR"

		err := os.Setenv(existingKey, "exists")
		require.NoError(t, err)
		defer os.Unsetenv(existingKey)

		os.Unsetenv(nonExistingKey) // Ensure it doesn't exist

		existingResult := GetString(existingKey, "default")
		nonExistingResult := GetString(nonExistingKey, "default")

		assert.Equal(t, "exists", existingResult)
		assert.Equal(t, "default", nonExistingResult)
	})
}

// Edge case tests
func TestEdgeCases(t *testing.T) {
	t.Run("Very large integer", func(t *testing.T) {
		key := "LARGE_INT_VAR"
		largeInt := "2147483647" // Max int32

		err := os.Setenv(key, largeInt)
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetInt(key, 0)

		assert.Equal(t, 2147483647, result)
	})

	t.Run("Very small integer", func(t *testing.T) {
		key := "SMALL_INT_VAR"
		smallInt := "-2147483648" // Min int32

		err := os.Setenv(key, smallInt)
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetInt(key, 0)

		assert.Equal(t, -2147483648, result)
	})

	t.Run("Unicode string", func(t *testing.T) {
		key := "UNICODE_VAR"
		unicode := "Hello 世界! 🌍"

		err := os.Setenv(key, unicode)
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetString(key, "default")

		assert.Equal(t, unicode, result)
	})

	t.Run("String with newlines and special characters", func(t *testing.T) {
		key := "SPECIAL_CHARS_VAR"
		special := "line1\nline2\ttab\r\nwindows"

		err := os.Setenv(key, special)
		require.NoError(t, err)
		defer os.Unsetenv(key)

		result := GetString(key, "default")

		assert.Equal(t, special, result)
	})
}

// Benchmark tests
func BenchmarkGetString(b *testing.B) {
	key := "BENCH_STRING_VAR"
	err := os.Setenv(key, "benchmark_value")
	require.NoError(b, err)
	defer os.Unsetenv(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetString(key, "default")
	}
}

func BenchmarkGetStringNotExists(b *testing.B) {
	key := "NON_EXISTENT_BENCH_VAR"
	os.Unsetenv(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetString(key, "default")
	}
}

func BenchmarkGetInt(b *testing.B) {
	key := "BENCH_INT_VAR"
	err := os.Setenv(key, "12345")
	require.NoError(b, err)
	defer os.Unsetenv(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetInt(key, 0)
	}
}

func BenchmarkGetBool(b *testing.B) {
	key := "BENCH_BOOL_VAR"
	err := os.Setenv(key, "true")
	require.NoError(b, err)
	defer os.Unsetenv(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetBool(key, false)
	}
}
