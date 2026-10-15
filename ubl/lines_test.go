package ubl_test

import (
	"fmt"
	"testing"

	"github.com/invopop/gobl"
	zatcaubl "github.com/invopop/gobl.sa.zatca/ubl"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ksa11 is a line's VAT amount (KSA-11) and the line amount including VAT
// (KSA-12), read back off a converted document.
type ksa11 struct {
	net     num.Amount
	vat     num.Amount
	withVAT num.Amount
}

// exportKSA11 converts the invoice and reads each line's KSA-11 and KSA-12,
// failing the test when a line omits them, which BR-KSA-52 and BR-KSA-53
// forbid.
func exportKSA11(t *testing.T, inv *bill.Invoice) []ksa11 {
	t.Helper()
	env, err := gobl.Envelop(inv)
	require.NoError(t, err)
	doc, err := goblubl.ExportInvoice(env, goblubl.WithFormat(zatcaubl.FormatZATCA))
	require.NoError(t, err)

	require.NotEmpty(t, doc.InvoiceLines, "a ZATCA document always carries invoice lines")
	out := make([]ksa11, 0, len(doc.InvoiceLines))
	for i, l := range doc.InvoiceLines {
		require.Len(t, l.TaxTotal, 1, "line %d must state its VAT amount (BR-KSA-52)", i+1)
		require.NotNil(t, l.TaxTotal[0].RoundingAmount, "line %d must state its amount with VAT (BR-KSA-53)", i+1)
		out = append(out, ksa11{
			net:     amt(t, l.LineExtensionAmount.Value),
			vat:     amt(t, l.TaxTotal[0].TaxAmount.Value),
			withVAT: amt(t, l.TaxTotal[0].RoundingAmount.Value),
		})
	}
	return out
}

// ksaInvoice builds a minimal simplified invoice carrying the given lines,
// optionally with tax-inclusive prices.
func ksaInvoice(pricesInclude bool, lines ...*bill.Line) *bill.Invoice {
	inv := &bill.Invoice{
		Regime:    tax.WithRegime("SA"),
		Addons:    tax.WithAddons("sa-zatca-v1"),
		Tags:      tax.WithTags(tax.TagSimplified),
		Currency:  currency.SAR,
		IssueDate: cal.MakeDate(2024, 4, 2),
		IssueTime: cal.NewTime(17, 45, 0),
		Type:      bill.InvoiceTypeStandard,
		Code:      "INV-1",
		Supplier: &org.Party{
			Name:       "Maximum Speed Tech Supply LTD",
			TaxID:      &tax.Identity{Country: "SA", Code: "399999999900003"},
			Identities: []*org.Identity{{Type: "CRN", Code: "1010010000"}},
			Addresses: []*org.Address{{
				Street: "Prince Sultan", StreetExtra: "Al-Murabba", Number: "2322",
				Locality: "Riyadh", Region: "Riyadh", Code: "23333", Country: "SA",
			}},
		},
		Lines: lines,
	}
	if pricesInclude {
		inv.Tax = &bill.Tax{PricesInclude: tax.CategoryVAT}
	}
	return inv
}

// TestKSA11ReproducesTaxInclusivePrice is the regression guard for the first
// of the two line VAT faults: when prices include VAT, GOBL divides each line
// down to a net amount that cannot be multiplied back up to the price charged.
// KSA-11 must therefore state the VAT the price contained, so that KSA-12
// reproduces that price to the cent.
func TestKSA11ReproducesTaxInclusivePrice(t *testing.T) {
	// Prices that divide by 1.15 into a net sitting the wrong side of a
	// half-cent; each one shipped a cent out before the fix.
	prices := []string{
		"1002.30", "10.00", "333.33", "0.05", "999.99", "0.01", "0.02", "0.03",
		"1.15", "2.30", "7.77", "19.99", "23.00", "45.45", "99.95", "100.00",
		"123.45", "250.01", "555.55", "1234.56", "9999.99", "0.07", "0.13",
	}
	for _, price := range prices {
		t.Run(price, func(t *testing.T) {
			charged := amt(t, price)
			res := exportKSA11(t, ksaInvoice(true, vatLine(t, price, standardVAT("15%"))))
			require.Len(t, res, 1)
			l := res[0]

			assert.Equal(t, 0, l.withVAT.Compare(charged),
				"KSA-12 (%s) must be the price charged (%s)", l.withVAT, charged)
			assert.Equal(t, 0, l.net.Add(l.vat).Compare(l.withVAT),
				"KSA-12 (%s) must be BT-131 (%s) + KSA-11 (%s) [BR-KSA-51]", l.withVAT, l.net, l.vat)
			assertTwoDecimals(t, l.vat.String(), "KSA-11 [BR-KSA-DEC-03]")
			assertTwoDecimals(t, l.withVAT.String(), "KSA-12 [BR-KSA-DEC-04]")
		})
	}
}

// TestKSA11SweepsEveryCentAtStandardRate converts every price from 0.01 to
// 25.00 with tax-inclusive prices and checks the line still charges exactly
// what it quoted. A rate applied to the rounded net drifts on roughly one
// price in seven, so a sweep catches what a handful of fixtures cannot.
func TestKSA11SweepsEveryCentAtStandardRate(t *testing.T) {
	for cents := 1; cents <= 2500; cents++ {
		price := num.MakeAmount(int64(cents), 2)
		res := exportKSA11(t, ksaInvoice(true, vatLine(t, price.String(), standardVAT("15%"))))
		require.Len(t, res, 1)
		l := res[0]
		require.Equal(t, 0, l.withVAT.Compare(price),
			"price %s: KSA-12 (%s) must be the price charged", price, l.withVAT)
		require.Equal(t, 0, l.net.Add(l.vat).Compare(l.withVAT),
			"price %s: KSA-12 (%s) must be BT-131 (%s) + KSA-11 (%s)", price, l.withVAT, l.net, l.vat)
	}
}

// TestKSA11UntaxedCategoriesStateZero is the regression guard for the second
// fault: exempt, zero-rated and out-of-scope lines carry no percentage, so
// the VAT amount could not be calculated and was omitted entirely, failing
// BR-KSA-52 and BR-KSA-53.
func TestKSA11UntaxedCategoriesStateZero(t *testing.T) {
	tests := []struct {
		name     string
		key      cbc.Key
		category cbc.Code
		vatex    cbc.Code
	}{
		{"zero rated", tax.KeyZero, "Z", "VATEX-SA-35"},
		{"exempt", tax.KeyExempt, "E", "VATEX-SA-29"},
		{"outside scope", cbc.KeyEmpty, "O", "VATEX-SA-OOS"},
	}
	for _, tt := range tests {
		for _, pricesInclude := range []bool{false, true} {
			name := tt.name
			if pricesInclude {
				name += ", prices inclusive"
			}
			t.Run(name, func(t *testing.T) {
				combo := untaxedVAT(tt.key, tt.category, tt.vatex)
				res := exportKSA11(t, ksaInvoice(pricesInclude, vatLine(t, "200.00", combo)))
				require.Len(t, res, 1)
				l := res[0]

				assert.Equal(t, "0.00", l.vat.String(),
					"an untaxed line states a zero VAT amount rather than omitting it (BR-KSA-52)")
				assert.Equal(t, 0, l.withVAT.Compare(l.net),
					"KSA-12 (%s) must equal BT-131 (%s) when no VAT is charged", l.withVAT, l.net)
				assert.Equal(t, 0, l.withVAT.Compare(amt(t, "200.00")),
					"an untaxed line charges exactly its price")
			})
		}
	}
}

// TestKSA11MixedCategories checks a document that mixes a rated line with
// untaxed ones, which is where the two faults met: the rated line
// needed the charged price and the untaxed ones needed a stated zero.
func TestKSA11MixedCategories(t *testing.T) {
	for _, pricesInclude := range []bool{false, true} {
		name := "prices exclusive"
		if pricesInclude {
			name = "prices inclusive"
		}
		t.Run(name, func(t *testing.T) {
			res := exportKSA11(t, ksaInvoice(pricesInclude,
				vatLine(t, "1002.30", standardVAT("15%")),
				vatLine(t, "50.00", untaxedVAT(tax.KeyZero, "Z", "VATEX-SA-35")),
				vatLine(t, "200.00", untaxedVAT(tax.KeyExempt, "E", "VATEX-SA-29")),
				vatLine(t, "300.00", untaxedVAT(cbc.KeyEmpty, "O", "VATEX-SA-OOS")),
			))
			require.Len(t, res, 4)

			for i, l := range res {
				assert.Equal(t, 0, l.net.Add(l.vat).Compare(l.withVAT),
					"line %d: KSA-12 must be BT-131 + KSA-11 [BR-KSA-51]", i+1)
			}
			for i, l := range res[1:] {
				assert.Equal(t, "0.00", l.vat.String(), "untaxed line %d states a zero VAT amount", i+2)
			}
			if pricesInclude {
				for i, want := range []string{"1002.30", "50.00", "200.00", "300.00"} {
					assert.Equal(t, 0, res[i].withVAT.Compare(amt(t, want)),
						"line %d charges exactly its price", i+1)
				}
			}
		})
	}
}

// TestKSA11CreditNoteLines checks the fix survives the rework that moves a
// credit note's lines out of CreditNoteLines, since ZATCA writes every
// document as an invoice.
func TestKSA11CreditNoteLines(t *testing.T) {
	inv := ksaInvoice(true,
		vatLine(t, "1002.30", standardVAT("15%")),
		vatLine(t, "10.00", standardVAT("15%")),
	)
	inv.Type = bill.InvoiceTypeCreditNote
	inv.Preceding = []*org.DocumentRef{{
		Code: "INV-0", IssueDate: cal.NewDate(2024, 3, 1), Reason: "Correction",
	}}

	res := exportKSA11(t, inv)
	require.Len(t, res, 2)
	for i, want := range []string{"1002.30", "10.00"} {
		assert.Equal(t, 0, res[i].withVAT.Compare(amt(t, want)),
			"credit note line %d charges exactly its price", i+1)
	}
}

// TestKSA11LineCurrency checks a line priced in another currency is rounded
// to that currency rather than the document's.
func TestKSA11LineCurrency(t *testing.T) {
	line := vatLine(t, "100.00", standardVAT("15%"))
	line.Item.Currency = currency.USD
	inv := ksaInvoice(false, line)
	inv.ExchangeRates = []*currency.ExchangeRate{
		{From: currency.USD, To: currency.SAR, Amount: amt(t, "3.75")},
	}

	res := exportKSA11(t, inv)
	require.Len(t, res, 1)
	assert.Equal(t, 0, res[0].net.Add(res[0].vat).Compare(res[0].withVAT),
		"KSA-12 must be BT-131 + KSA-11 whatever the line currency")
	assertTwoDecimals(t, res[0].vat.String(), "KSA-11 in a foreign line currency")
}

// TestKSA11ZeroAmountLine checks a line that charges nothing still states its
// amounts rather than omitting them.
func TestKSA11ZeroAmountLine(t *testing.T) {
	for _, pricesInclude := range []bool{false, true} {
		t.Run(fmt.Sprintf("pricesInclude=%v", pricesInclude), func(t *testing.T) {
			res := exportKSA11(t, ksaInvoice(pricesInclude,
				vatLine(t, "0.00", standardVAT("15%")),
				vatLine(t, "10.00", standardVAT("15%")),
			))
			require.Len(t, res, 2)
			assert.Equal(t, "0.00", res[0].vat.String(), "a zero line states a zero VAT amount")
			assert.Equal(t, "0.00", res[0].withVAT.String(), "a zero line charges nothing")
		})
	}
}

// TestKSA11ReducedRate checks the fix is not tied to the 15% standard rate.
func TestKSA11ReducedRate(t *testing.T) {
	for _, price := range []string{"1002.30", "10.00", "333.33", "0.03"} {
		t.Run(price, func(t *testing.T) {
			res := exportKSA11(t, ksaInvoice(true, vatLine(t, price, standardVAT("5%"))))
			require.Len(t, res, 1)
			assert.Equal(t, 0, res[0].withVAT.Compare(amt(t, price)),
				"KSA-12 must be the price charged at a reduced rate too")
		})
	}
}

// TestKSA11LineDiscountsAndCharges checks the lines whose total is not simply
// the unit price: the accuracy a tax-inclusive price leaves behind sits on the
// price alone, so a line discount or charge has to be applied on top of it
// rather than to the rounded net.
func TestKSA11LineDiscountsAndCharges(t *testing.T) {
	tests := []struct {
		name    string
		line    *bill.Line
		charged string
	}{
		{
			name: "percentage discount",
			line: &bill.Line{
				Quantity:  num.MakeAmount(3, 0),
				Item:      &org.Item{Name: "Service", Price: num.NewAmount(100045, 2)},
				Discounts: []*bill.LineDiscount{{Percent: num.NewPercentage(10, 2)}},
				Taxes:     tax.Set{standardVAT("15%")},
			},
			charged: "2701.21",
		},
		{
			name: "fixed charge",
			line: &bill.Line{
				Quantity: num.MakeAmount(1, 0),
				Item:     &org.Item{Name: "Service", Price: num.NewAmount(33333, 2)},
				Charges:  []*bill.LineCharge{{Amount: num.MakeAmount(500, 2)}},
				Taxes:    tax.Set{standardVAT("15%")},
			},
			charged: "338.33",
		},
		{
			name: "quantity above one",
			line: &bill.Line{
				Quantity: num.MakeAmount(7, 0),
				Item:     &org.Item{Name: "Service", Price: num.NewAmount(1230, 2)},
				Taxes:    tax.Set{standardVAT("15%")},
			},
			charged: "86.10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := exportKSA11(t, ksaInvoice(true, tt.line))
			require.Len(t, res, 1)
			l := res[0]

			assert.Equal(t, 0, l.withVAT.Compare(amt(t, tt.charged)),
				"KSA-12 (%s) must be the amount charged (%s)", l.withVAT, tt.charged)
			assert.Equal(t, 0, l.net.Add(l.vat).Compare(l.withVAT),
				"KSA-12 (%s) must be BT-131 (%s) + KSA-11 (%s) [BR-KSA-51]", l.withVAT, l.net, l.vat)
			assertTwoDecimals(t, l.vat.String(), "KSA-11 [BR-KSA-DEC-03]")
			assertTwoDecimals(t, l.withVAT.String(), "KSA-12 [BR-KSA-DEC-04]")
		})
	}
}
