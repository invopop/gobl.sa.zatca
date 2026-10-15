package zatca_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocumentTypeScenarios covers the UNTDID document types ZATCA restates
// over EN 16931, as BR-KSA-05 accepts only 388, 386, 383 and 381. Tags that
// EN 16931 gives their own code, such as self-billing and factoring, keep
// the plain code in ZATCA.
func TestDocumentTypeScenarios(t *testing.T) {
	selfBilled := func(inv *bill.Invoice) *bill.Invoice {
		inv.Customer.Identities = []*org.Identity{
			{Type: "CRN", Code: "1234567890"},
		}
		return inv
	}

	tests := []struct {
		name string
		inv  *bill.Invoice
		tags []cbc.Key
		code cbc.Code
	}{
		{"tax invoice", validStandardInvoice(), nil, "388"},
		{"self-billed tax invoice", selfBilled(validStandardInvoice()), []cbc.Key{tax.TagSelfBilled}, "388"},
		{"prepayment invoice", validStandardInvoice(), []cbc.Key{tax.TagPrepayment}, "386"},
		{"simplified prepayment invoice", validSimplifiedInvoice(), []cbc.Key{tax.TagPrepayment}, "386"},
		{"self-billed prepayment invoice", selfBilled(validStandardInvoice()), []cbc.Key{tax.TagSelfBilled, tax.TagPrepayment}, "386"},
		{"credit note", validCreditNote(), nil, "381"},
		{"self-billed credit note", selfBilled(validCreditNote()), []cbc.Key{tax.TagSelfBilled}, "381"},
		{"factored credit note", validCreditNote(), []cbc.Key{tax.TagFactoring}, "381"},
		{"self-billed factored credit note", selfBilled(validCreditNote()), []cbc.Key{tax.TagSelfBilled, tax.TagFactoring}, "381"},
		{"debit note", validDebitNote(), nil, "383"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := tt.inv
			inv.SetTags(append(inv.GetTags(), tt.tags...)...)
			require.NoError(t, inv.Calculate())

			assert.Equal(t, tt.code, inv.Tax.Ext.Get(untdid.ExtKeyDocumentType))
			assert.NoError(t, rules.Validate(inv))
		})
	}
}
