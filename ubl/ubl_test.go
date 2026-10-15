package ubl_test

import (
	"bytes"
	"testing"

	zatca "github.com/invopop/gobl.sa.zatca/addon"
	zatcaubl "github.com/invopop/gobl.sa.zatca/ubl"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	ctx := convert.FormatFor(zatcaubl.KeyZATCA)
	require.NotNil(t, ctx)
	assert.Equal(t, cbc.Key("ubl"), ctx.Syntax)
	assert.Contains(t, ctx.Countries, l10n.SA)

	keys := make([]cbc.Key, 0)
	for _, c := range convert.FormatsFor("SA") {
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
	assert.Equal(t, zatcaubl.KeyZATCA, out.Format.Key)
	assert.Contains(t, string(out.Data), "<Invoice", "credit notes are written as invoices")

	ctx, err := convert.Detect(out.Data)
	require.NoError(t, err)
	assert.Equal(t, zatcaubl.KeyZATCA, ctx.Key, "detected again")
}

// TestImportDocumentType covers BT-3 against the codes BR-KSA-05 accepts: each
// must come back as the type and tags that produce the same code again.
func TestImportDocumentType(t *testing.T) {
	tests := []struct {
		name string
		file string
		from string
		to   string
		typ  cbc.Key
		tags []cbc.Key
		code cbc.Code
	}{
		{"tax invoice", "zatca/standard-invoice.xml", "", "", bill.InvoiceTypeStandard, nil, "388"},
		{"prepayment invoice", "zatca/standard-prepayment.xml", "", "", bill.InvoiceTypeStandard, []cbc.Key{tax.TagPrepayment}, "386"},
		{"simplified prepayment invoice", "zatca/simplified-invoice.xml", `name="0200000">388<`, `name="0200000">386<`, bill.InvoiceTypeStandard, []cbc.Key{tax.TagPrepayment, tax.TagSimplified}, "386"},
		{"credit note", "zatca/standard-credit-note.xml", "", "", bill.InvoiceTypeCreditNote, nil, "381"},
		{"self-billed credit note", "zatca/self-billed-credit-note.xml", "", "", bill.InvoiceTypeCreditNote, []cbc.Key{tax.TagSelfBilled}, "381"},
		{"debit note", "zatca/standard-debit-note.xml", "", "", bill.InvoiceTypeDebitNote, nil, "383"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := importWithTypeCode(t, tt.file, tt.from, tt.to)

			assert.Equal(t, tt.typ, inv.Type)
			assert.Equal(t, tt.tags, inv.GetTags())
			assert.Equal(t, tt.code, inv.Tax.Ext.Get(untdid.ExtKeyDocumentType))
			assert.NoError(t, rules.Validate(inv))
		})
	}

	// A code ZATCA does not accept is kept as stated, so BR-KSA-05 reports it
	// instead of the invoice quietly becoming a 388 or a 381.
	for _, tt := range []struct{ file, from, code string }{
		{"zatca/standard-invoice.xml", `name="0100000">388<`, "380"},
		{"zatca/standard-invoice.xml", `name="0100000">388<`, "389"},
		{"zatca/standard-invoice.xml", `name="0100000">388<`, "393"},
		{"zatca/standard-credit-note.xml", `name="0100000">381<`, "396"},
		{"zatca/standard-credit-note.xml", `name="0100000">381<`, "261"},
	} {
		t.Run(tt.code+" is reported", func(t *testing.T) {
			inv := importWithTypeCode(t, tt.file, tt.from, `name="0100000">`+tt.code+`<`)

			assert.Equal(t, bill.InvoiceTypeOther, inv.Type)
			assert.Equal(t, cbc.Code(tt.code), inv.Tax.Ext.Get(untdid.ExtKeyDocumentType))
			assert.ErrorContains(t, rules.Validate(inv), "BR-KSA-05")
		})
	}
}

// importWithTypeCode imports a UBL example with its BT-3 element swapped,
// unless from is empty.
func importWithTypeCode(t *testing.T, name, from, to string) *bill.Invoice {
	t.Helper()
	data, err := testLoadXML(name)
	require.NoError(t, err)
	if from != "" {
		swapped := bytes.Replace(data, []byte(from), []byte(to), 1)
		require.NotEqual(t, string(data), string(swapped), "BT-3 was not replaced")
		data = swapped
	}
	doc, err := goblubl.Decode(data)
	require.NoError(t, err)
	env, err := goblubl.Import(doc)
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	return inv
}
