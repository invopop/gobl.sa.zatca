package zatca_test

import (
	"testing"

	"github.com/invopop/gobl"
	zatca "github.com/invopop/gobl.sa.zatca"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lineTaxTest builds a simplified invoice whose lines carry the given prices
// and tax combos, so the line VAT amounts can be checked in isolation.
func lineTaxTest(t *testing.T, pricesInclude bool, lines ...*bill.Line) *zatca.Invoice {
	t.Helper()

	inv := &bill.Invoice{
		Regime:    tax.WithRegime("SA"),
		Addons:    tax.WithAddons("sa-zatca-v1"),
		Currency:  currency.SAR,
		IssueDate: cal.MakeDate(2024, 4, 2),
		IssueTime: cal.NewTime(17, 45, 0),
		Type:      bill.InvoiceTypeStandard,
		Code:      "INCL-001",
		Tags:      tax.WithTags(tax.TagSimplified),
		Supplier: &org.Party{
			Name:       "Maximum Speed Tech Supply LTD",
			TaxID:      &tax.Identity{Country: "SA", Code: "399999999900003"},
			Identities: []*org.Identity{{Type: "CRN", Code: "1010010000"}},
			Addresses: []*org.Address{{
				Street: "Prince Sultan", StreetExtra: "Al-Murabba", Number: "2322",
				Locality: "Riyadh", Code: "23333", Country: "SA",
			}},
		},
		Payment: &bill.PaymentDetails{
			Instructions: &pay.Instructions{Key: pay.MeansKeyCreditTransfer},
			Terms:        &pay.Terms{Notes: "Payment due within 30 days"},
		},
		Lines: lines,
	}
	if pricesInclude {
		inv.Tax = &bill.Tax{PricesInclude: tax.CategoryVAT}
	}

	env, err := gobl.Envelop(inv)
	require.NoError(t, err)

	doc, err := zatca.ConvertInvoice(env)
	require.NoError(t, err)
	return doc
}

func standardLine(price string) *bill.Line {
	amount := num.MakeAmount(0, 0)
	if v, err := num.AmountFromString(price); err == nil {
		amount = v
	}
	return &bill.Line{
		Quantity: num.MakeAmount(1, 0),
		Item:     &org.Item{Name: "Service", Price: &amount},
		Taxes:    tax.Set{{Category: tax.CategoryVAT, Percent: num.NewPercentage(15, 2)}},
	}
}

// TestLineTaxWithPricesIncludingVAT pins the line VAT amount (KSA-11) a price
// that includes tax produces. Dividing the tax out at the currency's precision
// and then taking 15% of the rounded net gives 130.74 on the first line, a cent
// more than the invoice charges, and a line amount with VAT (KSA-12) of 1002.31
// against a price of 1002.30.
func TestLineTaxWithPricesIncludingVAT(t *testing.T) {
	doc := lineTaxTest(t, true, standardLine("1002.30"), standardLine("500.00"), standardLine("333.33"))

	require.Len(t, doc.InvoiceLines, 3)
	expected := []struct{ net, vat, gross string }{
		{"871.57", "130.73", "1002.30"},
		{"434.78", "65.22", "500.00"},
		{"289.85", "43.48", "333.33"},
	}
	for i, want := range expected {
		line := doc.InvoiceLines[i]
		assert.Equal(t, want.net, line.LineExtensionAmount.Value, "line %d net amount (BT-131)", i+1)
		require.Len(t, line.TaxTotal, 1)
		assert.Equal(t, want.vat, line.TaxTotal[0].TaxAmount.Value, "line %d VAT amount (KSA-11)", i+1)
		require.NotNil(t, line.TaxTotal[0].RoundingAmount)
		assert.Equal(t, want.gross, line.TaxTotal[0].RoundingAmount.Value, "line %d amount with VAT (KSA-12)", i+1)
	}

	// The line VAT amounts must still add up to the document's own total.
	require.NotEmpty(t, doc.TaxTotal)
	assert.Equal(t, "239.43", doc.TaxTotal[0].TaxAmount.Value)
	assert.Equal(t, "1835.63", doc.LegalMonetaryTotal.TaxInclusiveAmount.Value)
	assert.Nil(t, doc.LegalMonetaryTotal.PayableRoundingAmount, "the payable must survive the division untouched")
}

// TestLineTaxWithoutPercent covers the lines that carry no VAT rate at all:
// BR-KSA-52 and BR-KSA-53 still require both amounts, so they are stated as
// zero rather than left out.
func TestLineTaxWithoutPercent(t *testing.T) {
	untaxed := func(category, vatex cbc.Code) *bill.Line {
		amount := num.MakeAmount(20000, 2)
		return &bill.Line{
			Quantity: num.MakeAmount(1, 0),
			Item:     &org.Item{Name: "Service", Price: &amount},
			Taxes: tax.Set{{
				Category: tax.CategoryVAT,
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyTaxCategory: category,
					cef.ExtKeyVATEX:          vatex,
				}),
			}},
		}
	}
	doc := lineTaxTest(t, false, untaxed("E", "VATEX-SA-29"), untaxed("O", "VATEX-SA-OOS"))

	require.Len(t, doc.InvoiceLines, 2)
	for i, line := range doc.InvoiceLines {
		require.Len(t, line.TaxTotal, 1, "line %d must state its VAT amount", i+1)
		assert.Equal(t, "0.00", line.TaxTotal[0].TaxAmount.Value)
		require.NotNil(t, line.TaxTotal[0].RoundingAmount)
		assert.Equal(t, "200.00", line.TaxTotal[0].RoundingAmount.Value)
	}
}
