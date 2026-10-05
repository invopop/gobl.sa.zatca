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

const (
	jsonPattern = "*.json"
	xmlPattern  = "*.xml"
)

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

// convertCases pairs every envelope under test/data/convert with the ZATCA XML
// it converts to. The envelopes are the shipped examples, copied in when the
// goldens are updated so each stage of the chain keeps its own input beside its
// output: TestExamples checks the examples as GOBL, and without this nothing
// would check them as ZATCA, so one could ship as invalid.
func convertCases(t *testing.T) []convertCase {
	t.Helper()
	regenerateExamples(t)
	if *update {
		copyStageInput(t, getExamplePath(), jsonPattern, getConvertPath())
	}

	found, err := filepath.Glob(filepath.Join(getConvertPath(), jsonPattern))
	require.NoError(t, err)
	require.NotEmpty(t, found, "no envelopes found in %s", getConvertPath())

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

// copyStageInput refreshes a stage's inputs from what the previous stage
// produced. Each stage reads files committed beside its own goldens rather than
// reaching into the one before it, so a fixture directory shows what went in
// as well as what came out.
func copyStageInput(t *testing.T, from, pattern, to string) {
	t.Helper()
	sources, err := filepath.Glob(filepath.Join(from, pattern))
	require.NoError(t, err)
	require.NotEmpty(t, sources, "nothing to copy from %s", from)
	require.NoError(t, os.MkdirAll(to, 0o755))

	keep := make(map[string]bool, len(sources))
	for _, src := range sources {
		data, err := os.ReadFile(src)
		require.NoError(t, err)
		name := filepath.Base(src)
		keep[name] = true
		require.NoError(t, os.WriteFile(filepath.Join(to, name), data, 0o644))
	}
	pruneStale(t, to, pattern, keep)
}

// pruneStale removes files in dir that this run did not produce, so a document
// dropped upstream stops being carried by every stage below it.
func pruneStale(t *testing.T, dir, pattern string, keep map[string]bool) {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(dir, pattern))
	require.NoError(t, err)
	for _, path := range found {
		if !keep[filepath.Base(path)] {
			require.NoError(t, os.Remove(path))
		}
	}
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
	cases := convertCases(t)
	for _, example := range cases {
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

	if *update {
		keep := make(map[string]bool, len(cases))
		for _, c := range cases {
			keep[filepath.Base(c.golden)] = true
		}
		pruneStale(t, filepath.Join(getConvertPath(), "out"), xmlPattern, keep)
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

	keep := make(map[string]bool, len(sources))
	for _, src := range sources {
		data, err := os.ReadFile(src)
		require.NoError(t, err)
		out, err := examples.Convert(data, examples.IsEnvelope(src))
		require.NoError(t, err, "converting %s", src)
		golden := examples.GoldenPath(src)
		keep[filepath.Base(golden)] = true
		require.NoError(t, os.MkdirAll(filepath.Dir(golden), 0o755))
		require.NoError(t, os.WriteFile(golden, out, 0o644))
	}
	pruneStale(t, getExamplePath(), jsonPattern, keep)
}
