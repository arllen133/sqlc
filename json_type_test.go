package sqlc

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSON tests the JSON[T] generic type
func TestJSON(t *testing.T) {
	type Metadata struct {
		Name  string   `json:"name"`
		Tags  []string `json:"tags"`
		Count int      `json:"count"`
	}

	t.Run("Value", func(t *testing.T) {
		j := JSON[Metadata]{
			Data: Metadata{
				Name:  "test",
				Tags:  []string{"a", "b"},
				Count: 42,
			},
		}

		val, err := j.Value()
		require.NoError(t, err)

		// Value should return JSON bytes
		bytes, ok := val.([]byte)
		require.True(t, ok, "expected []byte")

		var parsed Metadata
		err = json.Unmarshal(bytes, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "test", parsed.Name)
		assert.Equal(t, []string{"a", "b"}, parsed.Tags)
		assert.Equal(t, 42, parsed.Count)
	})

	t.Run("Scan from []byte", func(t *testing.T) {
		var j JSON[Metadata]
		input := []byte(`{"name":"scanned","tags":["x","y"],"count":100}`)

		err := j.Scan(input)
		require.NoError(t, err)

		assert.Equal(t, "scanned", j.Data.Name)
		assert.Equal(t, []string{"x", "y"}, j.Data.Tags)
		assert.Equal(t, 100, j.Data.Count)
	})

	t.Run("Scan from string", func(t *testing.T) {
		var j JSON[Metadata]
		input := `{"name":"from_string","tags":[],"count":0}`

		err := j.Scan(input)
		require.NoError(t, err)

		assert.Equal(t, "from_string", j.Data.Name)
	})

	t.Run("Scan from nil", func(t *testing.T) {
		var j JSON[Metadata]
		j.Data.Name = "preset"

		err := j.Scan(nil)
		require.NoError(t, err)

		// After scanning nil, Data should be zero value
		assert.Equal(t, "", j.Data.Name)
	})

	t.Run("Scan unsupported type", func(t *testing.T) {
		var j JSON[Metadata]
		err := j.Scan(12345)
		assert.Error(t, err)
	})

	t.Run("Implements driver.Valuer", func(t *testing.T) {
		var j any = JSON[Metadata]{}
		_, ok := j.(driver.Valuer)
		assert.True(t, ok, "JSON[T] should implement driver.Valuer")
	})
}

// TestJSONNested tests nested JSON structures
func TestJSONNested(t *testing.T) {
	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	t.Run("Nested struct", func(t *testing.T) {
		j := JSON[Person]{
			Data: Person{
				Name: "John",
				Address: Address{
					City:    "Tokyo",
					Country: "Japan",
				},
			},
		}

		val, err := j.Value()
		require.NoError(t, err)

		var j2 JSON[Person]
		err = j2.Scan(val)
		require.NoError(t, err)

		assert.Equal(t, "John", j2.Data.Name)
		assert.Equal(t, "Tokyo", j2.Data.Address.City)
		assert.Equal(t, "Japan", j2.Data.Address.Country)
	})
}
