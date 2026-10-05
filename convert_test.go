package zatca_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	zatca "github.com/invopop/gobl.sa.zatca"
	"github.com/invopop/gobl/pkg/examples"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jsonPattern = "*.json"

// getExamplePath is where TestExamples writes the calculated envelopes that
// feed the conversion.
func getExamplePath() string {
	return filepath.Join("examples", "out")
}

func getConvertPath() string {
	return filepath.Join("test", "data", "convert")
}

// convertCase is a GOBL envelope to convert, with the ZATCA XML it should
// produce.
type convertCase struct {
	name   string
	src    string
	golden string
}

// convertCases pairs every envelope TestExamples produces with the ZATCA XML it
// converts to. The examples are the only source: TestExamples checks them as
// GOBL, and without this nothing checks them as ZATCA, so one could ship as
// invalid. The XML goldens live under test/data/convert/out, which TestParse
// reads back in turn.
func convertCases(t *testing.T) []convertCase {
	t.Helper()
	regenerateExamples(t)

	found, err := filepath.Glob(filepath.Join(getExamplePath(), jsonPattern))
	require.NoError(t, err)
	require.NotEmpty(t, found, "no envelopes found in %s", getExamplePath())

	cases := make([]convertCase, 0, len(found))
	for _, src := range found {
		name := strings.TrimSuffix(filepath.Base(src), ".json")
		cases = append(cases, convertCase{
			name:   name,
			src:    src,
			golden: filepath.Join(getConvertPath(), "out", name+".xml"),
		})
	}
	return cases
}

// loadTestEnvelope loads a GOBL envelope from a JSON file path.
func loadTestEnvelope(t *testing.T, path string) *gobl.Envelope {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	env := new(gobl.Envelope)
	require.NoError(t, json.Unmarshal(data, env))
	return env
}

// TestConvert covers every fixture and shipped example, invoices, credit notes
// and debit notes alike.
func TestConvert(t *testing.T) {
	for _, example := range convertCases(t) {
		t.Run(example.name, func(t *testing.T) {
			env := loadTestEnvelope(t, example.src)

			doc, err := zatca.ConvertInvoice(env)
			require.NoError(t, err)

			data, err := zatca.Bytes(doc)
			require.NoError(t, err)

			if *update {
				require.NoError(t, os.MkdirAll(filepath.Dir(example.golden), 0o755))
				require.NoError(t, os.WriteFile(example.golden, data, 0644))
			}

			output, err := os.ReadFile(example.golden)
			assert.NoError(t, err)
			assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with -update flag.")
		})
	}
}

// regenerateExamples rebuilds the envelopes the conversion reads from, when the
// goldens are being updated. The chain has to run in order and Go does not:
// test files compile alphabetically, so TestConvert runs before TestExamples
// and would otherwise convert the envelopes as they stood before the run.
func regenerateExamples(t *testing.T) {
	t.Helper()
	if !*update {
		return
	}
	sources, err := examples.Sources("examples")
	require.NoError(t, err)
	require.NotEmpty(t, sources, "no example sources found")
	for _, src := range sources {
		data, err := os.ReadFile(src)
		require.NoError(t, err)
		out, err := examples.Convert(data, examples.IsEnvelope(src))
		require.NoError(t, err, "converting %s", src)
		golden := examples.GoldenPath(src)
		require.NoError(t, os.MkdirAll(filepath.Dir(golden), 0o755))
		require.NoError(t, os.WriteFile(golden, out, 0o644))
	}
}
