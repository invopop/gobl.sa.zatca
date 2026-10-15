package ubl

import (
	"slices"
	"testing"

	zatca "github.com/invopop/gobl.sa.zatca/addon"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocumentTypesMatchScenarios keeps the import table in step with the
// addon: each type and tags must produce their own code again.
func TestDocumentTypesMatchScenarios(t *testing.T) {
	for code, dt := range documentTypes {
		t.Run(code.String(), func(t *testing.T) {
			inv := &bill.Invoice{
				Regime:    tax.WithRegime("SA"),
				Addons:    tax.WithAddons(zatca.V1),
				Type:      dt.typ,
				Currency:  "SAR",
				IssueDate: cal.MakeDate(2024, 6, 15),
				Supplier:  &org.Party{Name: "Supplier", TaxID: &tax.Identity{Country: "SA"}},
				Lines: []*bill.Line{{
					Quantity: num.MakeAmount(1, 0),
					Item:     &org.Item{Name: "Item", Price: num.NewAmount(100, 0)},
					Taxes:    tax.Set{{Category: tax.CategoryVAT, Rate: "general"}},
				}},
			}
			inv.SetTags(slices.Clone(dt.tags)...)
			require.NoError(t, inv.Calculate())

			assert.Equal(t, code, inv.Tax.Ext.Get(untdid.ExtKeyDocumentType))
		})
	}
}

// TestImportDocumentTypeTagsNotShared guards the table against changes made to
// an imported invoice's tags: SetTags keeps the slice it is given.
func TestImportDocumentTypeTagsNotShared(t *testing.T) {
	// Restore the table should the copy regress, so other tests are unaffected.
	dt := documentTypes["386"]
	tags := slices.Clone(dt.tags)
	t.Cleanup(func() { dt.tags = tags; documentTypes["386"] = dt })

	prepayment := func() *bill.Invoice {
		out := &bill.Invoice{Tax: &bill.Tax{}}
		importDocumentType(&goblubl.IDType{Value: "386"}, out)
		return out
	}
	first := prepayment()
	require.Equal(t, []cbc.Key{tax.TagPrepayment}, first.GetTags())
	first.Tags.List[0] = tax.TagFactoring

	assert.Equal(t, []cbc.Key{tax.TagPrepayment}, prepayment().GetTags())
}

// TestImportInvoiceCreditNoteTypeCode reads KSA-2 from BT-3 whichever element
// carries it, so a CreditNote root keeps the flags it was sent with.
func TestImportInvoiceCreditNoteTypeCode(t *testing.T) {
	name := "0100001"
	in := &goblubl.Invoice{CreditNoteTypeCode: &goblubl.IDType{Value: "381", Name: &name}}
	out := &bill.Invoice{Tax: &bill.Tax{}}
	require.NoError(t, importInvoice(in, out))

	assert.Equal(t, cbc.Code(name), out.Tax.Ext.Get(zatca.ExtKeyInvoiceType))
	assert.Equal(t, bill.InvoiceTypeCreditNote, out.Type)
	assert.Equal(t, []cbc.Key{tax.TagSelfBilled}, out.GetTags())
}

// TestImportDocumentTypeKeepsOtherTags replaces only the tags the base import
// derived from BT-3: any other, such as bypass, stays on the invoice.
func TestImportDocumentTypeKeepsOtherTags(t *testing.T) {
	simplified := "0200000"
	tests := []struct {
		name string
		code string
		tags []cbc.Key
		want []cbc.Key
	}{
		{"bypass kept", "388", []cbc.Key{tax.TagBypass}, []cbc.Key{tax.TagBypass, tax.TagSimplified}},
		{"bypass kept on a prepayment", "386", []cbc.Key{tax.TagBypass}, []cbc.Key{tax.TagBypass, tax.TagPrepayment, tax.TagSimplified}},
		{"base BT-3 tag replaced", "389", []cbc.Key{tax.TagSelfBilled}, []cbc.Key{tax.TagSimplified}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bill.Invoice{Tax: &bill.Tax{}}
			out.SetTags(tt.tags...)
			importDocumentType(&goblubl.IDType{Value: tt.code, Name: &simplified}, out)

			assert.Equal(t, tt.want, out.GetTags())
		})
	}
}
