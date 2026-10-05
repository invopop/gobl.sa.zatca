package ubl_test

import (
	"testing"

	zatca "github.com/invopop/gobl.sa.zatca/addon"
	zatcaubl "github.com/invopop/gobl.sa.zatca/ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	ctx := convert.ContextFor(zatcaubl.KeyZATCA)
	require.NotNil(t, ctx)
	assert.Equal(t, cbc.Key("ubl"), ctx.Syntax)
	assert.Contains(t, ctx.Addons, zatca.V1)

	keys := make([]cbc.Key, 0)
	for _, c := range convert.ContextsFor("SA") {
		keys = append(keys, c.Key)
	}
	assert.Contains(t, keys, zatcaubl.KeyZATCA)
}

func TestDetectAndImport(t *testing.T) {
	data, err := testLoadXML("zatca/standard-invoice.xml")
	require.NoError(t, err)

	ctx, err := convert.Detect(data)
	require.NoError(t, err)
	assert.Equal(t, zatcaubl.KeyZATCA, ctx.Key)

	env, err := convert.Import(data)
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	assert.Contains(t, inv.GetAddons(), zatca.V1)
}

func TestExport(t *testing.T) {
	env := loadTestEnvelope(t, "zatca/standard-credit-note.json")
	out, err := convert.Export(env, zatcaubl.KeyZATCA)
	require.NoError(t, err)
	assert.Equal(t, zatcaubl.KeyZATCA, out.Context.Key)
	assert.Contains(t, string(out.Data), "<Invoice", "credit notes are written as invoices")

	ctx, err := convert.Detect(out.Data)
	require.NoError(t, err)
	assert.Equal(t, zatcaubl.KeyZATCA, ctx.Key, "detected again")
}
