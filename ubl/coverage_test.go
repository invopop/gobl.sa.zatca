package ubl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExamplesAreConverted checks that every shipped example is also a
// conversion fixture. TestExamples checks the examples as GOBL and stops
// there, so without this an example could ship that no ZATCA conversion or
// schematron run ever sees.
func TestExamplesAreConverted(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join("..", "examples", "out", "*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, examples)

	for _, src := range examples {
		name := filepath.Base(src)
		t.Run(name, func(t *testing.T) {
			_, err := os.Stat(filepath.Join("testdata", "convert", "zatca", name))
			assert.NoError(t, err, "example %s has no conversion fixture; copy it into testdata/convert/zatca", name)
		})
	}
}
