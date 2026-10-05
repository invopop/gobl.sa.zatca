package zatca_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	zatca "github.com/invopop/gobl.sa.zatca"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticHeadUUID replaces the envelope identifier a fresh parse mints, which is
// never reproducible across runs. The document's own UUID is read off the wire
// (BR-KSA-03) and has to survive the round trip, so it is left alone.
const staticHeadUUID uuid.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"

func getParsePath() string {
	return filepath.Join("test", "data", "parse")
}

// TestParse reads back every document TestConvert produced and compares the
// invoice against its golden envelope. Its input is the conversion's output, so
// the two tests together cover the round trip: a GOBL example out to ZATCA and
// back again.
func TestParse(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join(getConvertPath(), "out", "*.xml"))
	require.NoError(t, err)
	require.NotEmpty(t, examples, "no converted documents found")

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			data, err := os.ReadFile(example)
			require.NoError(t, err)

			doc, err := zatca.ParseInvoice(data)
			require.NoError(t, err)

			env, err := doc.Convert()
			require.NoError(t, err)

			env.Head.UUID = staticHeadUUID
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok, "document should be an invoice")
			require.NoError(t, env.Calculate())

			out, err := json.MarshalIndent(inv, "", "\t")
			require.NoError(t, err)

			outPath := filepath.Join(getParsePath(), "out", outName)
			if *update {
				golden, err := json.MarshalIndent(env, "", "\t")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(outPath, golden, 0644))
			}

			golden, err := os.ReadFile(outPath)
			require.NoError(t, err)
			expected := new(bill.Invoice)
			require.NoError(t, json.Unmarshal(goldenDoc(t, golden), expected))
			expectedData, err := json.MarshalIndent(expected, "", "\t")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(out), "Invoice should match the expected JSON. Update with -update flag.")
		})
	}
}

// goldenDoc pulls the invoice out of a golden envelope.
func goldenDoc(t *testing.T, data []byte) []byte {
	t.Helper()
	var env struct {
		Doc json.RawMessage `json:"doc"`
	}
	require.NoError(t, json.Unmarshal(data, &env))
	return env.Doc
}
