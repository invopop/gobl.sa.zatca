package zatca_test

import (
	"strings"
	"testing"

	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/norm"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeInvoice(t *testing.T) {
	t.Run("nil invoice does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			norm.Normalize((*bill.Invoice)(nil))
		})
	})

	t.Run("sets rounding to currency", func(t *testing.T) {
		inv := validStandardInvoice()
		norm.Normalize(inv)
		assert.Equal(t, tax.RoundingRuleCurrency, inv.Tax.Rounding)
	})

	t.Run("creates tax object when nil", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Tax = nil
		norm.Normalize(inv)
		require.NotNil(t, inv.Tax)
		assert.Equal(t, tax.RoundingRuleCurrency, inv.Tax.Rounding)
	})

	t.Run("creates issue time when nil", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.IssueTime = nil
		norm.Normalize(inv)
		require.NotNil(t, inv.IssueTime)
	})

	t.Run("line without VAT combo is skipped", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Lines = []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "No tax item",
					Price: num.NewAmount(50, 0),
				},
				Taxes: tax.Set{},
			},
		}
		assert.NotPanics(t, func() {
			norm.Normalize(inv)
		})
	})
}

func TestNormalizeInvoiceExemptionNotes(t *testing.T) {
	t.Run("exempt line gets tax note", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Lines = []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "Exempt item",
					Price: num.NewAmount(100, 0),
				},
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Key:      tax.KeyExempt,
						Ext: tax.ExtensionsOf(cbc.CodeMap{
							cef.ExtKeyVATEX:          "VATEX-SA-29",
							untdid.ExtKeyTaxCategory: en16931.TaxCategoryExempt,
						}),
					},
				},
			},
		}
		norm.Normalize(inv)
		require.Len(t, inv.Tax.Notes, 1)
		n := inv.Tax.Notes[0]
		assert.Equal(t, tax.CategoryVAT, n.Category)
		assert.Equal(t, tax.KeyExempt, n.Key)
		assert.Equal(t, "Financial services mentioned in Article 29 of the VAT Regulations", n.Text)
		assert.Equal(t, en16931.TaxCategoryExempt, n.Ext.Get(untdid.ExtKeyTaxCategory))
	})

	t.Run("zero-rated line gets tax note", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Lines = []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "Export item",
					Price: num.NewAmount(100, 0),
				},
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Key:      tax.KeyZero,
						Ext: tax.ExtensionsOf(cbc.CodeMap{
							cef.ExtKeyVATEX:          "VATEX-SA-32",
							untdid.ExtKeyTaxCategory: en16931.TaxCategoryZero,
						}),
					},
				},
			},
		}
		norm.Normalize(inv)
		require.Len(t, inv.Tax.Notes, 1)
		n := inv.Tax.Notes[0]
		assert.Equal(t, tax.CategoryVAT, n.Category)
		assert.Equal(t, tax.KeyZero, n.Key)
		assert.Equal(t, "Export of goods", n.Text)
		assert.Equal(t, en16931.TaxCategoryZero, n.Ext.Get(untdid.ExtKeyTaxCategory))
	})

	t.Run("outside-scope line gets tax note", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Lines = []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "OOS item",
					Price: num.NewAmount(100, 0),
				},
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Key:      tax.KeyOutsideScope,
						Ext: tax.ExtensionsOf(cbc.CodeMap{
							cef.ExtKeyVATEX:          "VATEX-SA-OOS",
							untdid.ExtKeyTaxCategory: en16931.TaxCategoryOutsideScope,
						}),
					},
				},
			},
		}
		norm.Normalize(inv)
		require.Len(t, inv.Tax.Notes, 1)
		n := inv.Tax.Notes[0]
		assert.Equal(t, tax.CategoryVAT, n.Category)
		assert.Equal(t, tax.KeyOutsideScope, n.Key)
		assert.Equal(t, "Reason is free text, to be provided by the taxpayer on a case-by-case basis", n.Text)
		assert.Equal(t, en16931.TaxCategoryOutsideScope, n.Ext.Get(untdid.ExtKeyTaxCategory))
	})

	t.Run("standard VAT line does not add note", func(t *testing.T) {
		inv := validStandardInvoice()
		norm.Normalize(inv)
		assert.Empty(t, inv.Tax.Notes)
	})

	t.Run("existing note for same category not duplicated", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Tax = &bill.Tax{
			Notes: []*tax.Note{
				{
					Category: tax.CategoryVAT,
					Key:      tax.KeyExempt,
					Text:     "Existing exemption note",
					Ext:      tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTaxCategory: en16931.TaxCategoryExempt}),
				},
			},
		}
		inv.Lines = []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "Exempt item",
					Price: num.NewAmount(100, 0),
				},
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Key:      tax.KeyExempt,
						Ext: tax.ExtensionsOf(cbc.CodeMap{
							cef.ExtKeyVATEX:          "VATEX-SA-29",
							untdid.ExtKeyTaxCategory: en16931.TaxCategoryExempt,
						}),
					},
				},
			},
		}
		norm.Normalize(inv)
		assert.Len(t, inv.Tax.Notes, 1)
		assert.Equal(t, "Existing exemption note", inv.Tax.Notes[0].Text)
	})

	t.Run("nil note in notes slice is skipped", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Tax = &bill.Tax{
			Notes: []*tax.Note{nil},
		}
		inv.Lines = []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "Exempt item",
					Price: num.NewAmount(100, 0),
				},
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Key:      tax.KeyExempt,
						Ext: tax.ExtensionsOf(cbc.CodeMap{
							cef.ExtKeyVATEX:          "VATEX-SA-29",
							untdid.ExtKeyTaxCategory: en16931.TaxCategoryExempt,
						}),
					},
				},
			},
		}
		norm.Normalize(inv)
		require.Len(t, inv.Tax.Notes, 1)
		assert.Equal(t, "Financial services mentioned in Article 29 of the VAT Regulations", inv.Tax.Notes[0].Text)
	})

	t.Run("unknown VATEX code does not add note", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Lines = []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "Unknown item",
					Price: num.NewAmount(100, 0),
				},
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Key:      tax.KeyExempt,
						Ext: tax.ExtensionsOf(cbc.CodeMap{
							cef.ExtKeyVATEX:          "VATEX-XX-99",
							untdid.ExtKeyTaxCategory: en16931.TaxCategoryExempt,
						}),
					},
				},
			},
		}
		norm.Normalize(inv)
		assert.Empty(t, inv.Tax.Notes)
	})

	t.Run("multiple lines with different categories get separate notes", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Lines = []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "Exempt item",
					Price: num.NewAmount(100, 0),
				},
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Key:      tax.KeyExempt,
						Ext: tax.ExtensionsOf(cbc.CodeMap{
							cef.ExtKeyVATEX:          "VATEX-SA-29",
							untdid.ExtKeyTaxCategory: en16931.TaxCategoryExempt,
						}),
					},
				},
			},
			{
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "Export item",
					Price: num.NewAmount(200, 0),
				},
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Key:      tax.KeyZero,
						Ext: tax.ExtensionsOf(cbc.CodeMap{
							cef.ExtKeyVATEX:          "VATEX-SA-32",
							untdid.ExtKeyTaxCategory: en16931.TaxCategoryZero,
						}),
					},
				},
			},
		}
		norm.Normalize(inv)
		require.Len(t, inv.Tax.Notes, 2)
		assert.Equal(t, en16931.TaxCategoryExempt, inv.Tax.Notes[0].Ext.Get(untdid.ExtKeyTaxCategory))
		assert.Equal(t, en16931.TaxCategoryZero, inv.Tax.Notes[1].Ext.Get(untdid.ExtKeyTaxCategory))
	})
}

// TestNormalizeTaxNotesFromCEF verifies that, for every ZATCA VATEX-SA
// exemption code, an invoice line carrying that code but no user-provided tax
// note is automatically given one whose text is sourced from the CEF VATEX
// extension definition. This guards against re-introducing a locally
// maintained code->reason map that could drift from the catalogue.
func TestNormalizeTaxNotesFromCEF(t *testing.T) {
	vatex := tax.ExtensionForKey(cef.ExtKeyVATEX)
	require.NotNil(t, vatex)

	var saCodes int
	for _, def := range vatex.Values {
		if !strings.HasPrefix(def.Code.String(), "VATEX-SA") {
			continue
		}
		saCodes++
		t.Run(def.Code.String(), func(t *testing.T) {
			inv := validStandardInvoice()
			inv.Tax = &bill.Tax{} // no user-provided notes
			inv.Lines = []*bill.Line{
				{
					Quantity: num.MakeAmount(1, 0),
					Item: &org.Item{
						Name:  "Exempt item",
						Price: num.NewAmount(100, 0),
					},
					Taxes: tax.Set{
						{
							Category: tax.CategoryVAT,
							Key:      tax.KeyExempt,
							Ext: tax.ExtensionsOf(cbc.CodeMap{
								cef.ExtKeyVATEX:          def.Code,
								untdid.ExtKeyTaxCategory: en16931.TaxCategoryExempt,
							}),
						},
					},
				},
			}
			norm.Normalize(inv)

			want := def.Name.In(i18n.EN)
			require.NotEmpty(t, want, "CEF definition must provide an English name")
			require.Len(t, inv.Tax.Notes, 1)
			assert.Equal(t, want, inv.Tax.Notes[0].Text)
		})
	}
	assert.Equal(t, 16, saCodes, "expected all ZATCA VATEX-SA codes to be exercised")
}

func TestBillDiscountRules(t *testing.T) {
	t.Run("discount with taxes is valid", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Discounts = []*bill.Discount{
			{
				Reason: "Loyalty discount",
				Amount: num.MakeAmount(50, 0),
				Taxes: tax.Set{
					{
						Category: tax.CategoryVAT,
						Rate:     tax.RateGeneral,
					},
				},
			},
		}
		require.NoError(t, inv.Calculate())
		assert.NoError(t, rules.Validate(inv))
	})

	t.Run("discount without taxes fails", func(t *testing.T) {
		inv := validStandardInvoice()
		inv.Discounts = []*bill.Discount{
			{
				Reason: "Loyalty discount",
				Amount: num.MakeAmount(50, 0),
			},
		}
		require.NoError(t, inv.Calculate())
		err := rules.Validate(inv)
		assert.ErrorContains(t, err, "taxes are required (BR-32)")
	})
}
