package zatca_test

import (
	"fmt"
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

// amt parses a decimal string into an amount, failing the test on bad input.
func amt(t *testing.T, s string) num.Amount {
	t.Helper()
	a, err := num.AmountFromString(s)
	require.NoError(t, err)
	return a
}

// vatLine builds a line at the given price carrying the given VAT combo.
func vatLine(t *testing.T, price string, combo *tax.Combo) *bill.Line {
	t.Helper()
	p := amt(t, price)
	return &bill.Line{
		Quantity: num.MakeAmount(1, 0),
		Item:     &org.Item{Name: "Service", Price: &p},
		Taxes:    tax.Set{combo},
	}
}

func standardVAT(percent string) *tax.Combo {
	p, _ := num.PercentageFromString(percent)
	return &tax.Combo{Category: tax.CategoryVAT, Key: tax.KeyStandard, Percent: &p}
}

// untaxedVAT builds a combo for a category that carries no rate: zero-rated,
// exempt, or outside the scope of VAT, each with the VATEX code ZATCA demands.
func untaxedVAT(key cbc.Key, category, vatex cbc.Code) *tax.Combo {
	return &tax.Combo{
		Category: tax.CategoryVAT,
		Key:      key,
		Ext: tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyTaxCategory: category,
			cef.ExtKeyVATEX:          vatex,
		}),
	}
}

// invoiceSpec describes one shape of document to convert and check.
type invoiceSpec struct {
	name          string
	simplified    bool
	docType       cbc.Key
	currency      currency.Code
	pricesInclude bool
	lines         []*bill.Line
	discounts     []*bill.Discount
	charges       []*bill.Charge
	rates         []*currency.ExchangeRate
}

func (s invoiceSpec) build(t *testing.T) *bill.Invoice {
	t.Helper()
	ccy := s.currency
	if ccy == currency.CodeEmpty {
		ccy = currency.SAR
	}
	docType := s.docType
	if docType == cbc.KeyEmpty {
		docType = bill.InvoiceTypeStandard
	}
	inv := &bill.Invoice{
		Regime:        tax.WithRegime("SA"),
		Addons:        tax.WithAddons("sa-zatca-v1"),
		Currency:      ccy,
		ExchangeRates: s.rates,
		IssueDate:     cal.MakeDate(2024, 4, 2),
		IssueTime:     cal.NewTime(17, 45, 0),
		Type:          docType,
		Code:          "INV-1",
		Supplier: &org.Party{
			Name:       "Maximum Speed Tech Supply LTD",
			TaxID:      &tax.Identity{Country: "SA", Code: "399999999900003"},
			Identities: []*org.Identity{{Type: "CRN", Code: "1010010000"}},
			Addresses: []*org.Address{{
				Street: "Prince Sultan", StreetExtra: "Al-Murabba", Number: "2322",
				Locality: "Riyadh", Region: "Riyadh", Code: "23333", Country: "SA",
			}},
		},
		Payment: &bill.PaymentDetails{
			Instructions: &pay.Instructions{Key: pay.MeansKeyCreditTransfer},
			Terms:        &pay.Terms{Notes: "Payment due within 30 days"},
		},
		Lines:     s.lines,
		Discounts: s.discounts,
		Charges:   s.charges,
	}
	if s.pricesInclude {
		inv.Tax = &bill.Tax{PricesInclude: tax.CategoryVAT}
	}
	if s.simplified {
		inv.Tags = tax.WithTags(tax.TagSimplified)
	} else {
		// A standard tax invoice needs a buyer and a delivery (BR-KSA-10, -72).
		inv.Customer = &org.Party{
			Name:       "Sample Consumer LLC",
			Identities: []*org.Identity{{Type: "TIN", Code: "123456789012345"}},
			Addresses: []*org.Address{{
				Street: "Olaya Street", StreetExtra: "Al Malaz", Number: "5678",
				Locality: "Riyadh", Region: "Riyadh", Code: "54321", Country: "SA",
			}},
		}
		inv.Delivery = &bill.DeliveryDetails{Date: cal.NewDate(2024, 4, 2)}
	}
	if docType.In(bill.InvoiceTypeCreditNote, bill.InvoiceTypeDebitNote) {
		inv.Preceding = []*org.DocumentRef{{
			Code: "INV-0", IssueDate: cal.NewDate(2024, 3, 1), Reason: "Correction",
		}}
	}
	return inv
}

// assertInvariants checks every arithmetic identity a ZATCA document has to
// hold, whatever shape it has. They are what the totals restatement must not
// break: the schematron checks most of them, and the rest are what the customer
// reads off the invoice.
func assertInvariants(t *testing.T, doc *zatca.Invoice, inv *bill.Invoice, gross []num.Amount) {
	t.Helper()
	zero := num.MakeAmount(0, 2)

	sumNet, sumVAT := zero, zero
	for i, line := range doc.InvoiceLines {
		require.Len(t, line.TaxTotal, 1, "line %d must state its VAT (BR-KSA-52)", i+1)
		require.NotNil(t, line.TaxTotal[0].RoundingAmount, "line %d must state its amount with VAT (BR-KSA-53)", i+1)
		net := amt(t, line.LineExtensionAmount.Value)
		vat := amt(t, line.TaxTotal[0].TaxAmount.Value)
		withVAT := amt(t, line.TaxTotal[0].RoundingAmount.Value)

		assert.Equal(t, 0, net.Add(vat).Compare(withVAT),
			"line %d: KSA-12 (%s) must be BT-131 (%s) + KSA-11 (%s) [BR-KSA-51]", i+1, withVAT, net, vat)
		assertTwoDecimals(t, line.LineExtensionAmount.Value, "line %d BT-131", i+1)
		assertTwoDecimals(t, line.TaxTotal[0].TaxAmount.Value, "line %d KSA-11 [BR-KSA-DEC-03]", i+1)
		assertTwoDecimals(t, line.TaxTotal[0].RoundingAmount.Value, "line %d KSA-12 [BR-KSA-DEC-04]", i+1)

		if gross != nil && i < len(gross) {
			assert.Equal(t, 0, withVAT.Compare(gross[i]),
				"line %d: KSA-12 (%s) must be the price charged (%s)", i+1, withVAT, gross[i])
		}
		sumNet = sumNet.Add(net)
		sumVAT = sumVAT.Add(vat)
	}

	mt := doc.LegalMonetaryTotal
	assert.Equal(t, 0, amt(t, mt.LineExtensionAmount.Value).Compare(sumNet),
		"BT-106 (%s) must be the sum of BT-131 (%s) [BR-CO-10]", mt.LineExtensionAmount.Value, sumNet)

	require.NotEmpty(t, doc.TaxTotal)
	docVAT := amt(t, doc.TaxTotal[0].TaxAmount.Value)
	subtotals := zero
	for _, st := range doc.TaxTotal[0].TaxSubtotal {
		subtotals = subtotals.Add(amt(t, st.TaxAmount.Value))
	}
	assert.Equal(t, 0, docVAT.Compare(subtotals),
		"BT-110 (%s) must be the sum of BT-117 (%s) [BR-CO-14]", docVAT, subtotals)

	exclusive := amt(t, mt.TaxExclusiveAmount.Value)
	inclusive := amt(t, mt.TaxInclusiveAmount.Value)
	assert.Equal(t, 0, exclusive.Add(docVAT).Compare(inclusive),
		"BT-112 (%s) must be BT-109 (%s) + BT-110 (%s) [BR-CO-15]", inclusive, exclusive, docVAT)

	require.NotNil(t, mt.PayableAmount)
	payable := amt(t, mt.PayableAmount.Value)
	expected := inclusive
	if mt.PrepaidAmount != nil {
		expected = expected.Subtract(amt(t, mt.PrepaidAmount.Value))
	}
	if mt.PayableRoundingAmount != nil {
		expected = expected.Add(amt(t, mt.PayableRoundingAmount.Value))
	}
	assert.Equal(t, 0, payable.Compare(expected),
		"BT-115 (%s) must be BT-112 - BT-113 + BT-114 (%s) [BR-CO-16]", payable, expected)

	// The amount payable is what the customer owes: no rework may move it.
	owed := inv.Totals.Payable
	if inv.Totals.Due != nil {
		owed = *inv.Totals.Due
	}
	assert.Equal(t, 0, payable.Compare(owed),
		"the amount payable (%s) must be what GOBL calculated (%s)", payable, owed)

	// The line VAT amounts are not required to sum to BT-110 and generally will
	// not: GOBL derives the document VAT from the summed base, where KSA-11 is
	// the VAT each price actually contains. No ZATCA rule compares the two.
	// What must hold is that they never drift more than a cent per line apart,
	// which would mean the two are measuring different things.
	// A document-level allowance or charge carries VAT of its own that no line
	// states, so only a document made of lines alone can be bounded this way.
	if len(doc.AllowanceCharge) == 0 {
		drift := abs(docVAT.Subtract(sumVAT))
		limit := num.MakeAmount(int64(len(doc.InvoiceLines)), 2)
		assert.LessOrEqual(t, drift.Compare(limit), 0,
			"BT-110 (%s) and the line VAT amounts (%s) drift by more than a cent a line", docVAT, sumVAT)
	}
}

// abs returns the amount without its sign.
func abs(a num.Amount) num.Amount {
	if a.Compare(num.MakeAmount(0, a.Exp())) < 0 {
		return a.Invert()
	}
	return a
}

// assertTwoDecimals checks the cap every ZATCA amount is held to (BR-KSA-DEC).
func assertTwoDecimals(t *testing.T, value, format string, args ...any) {
	t.Helper()
	if i := indexByte(value, '.'); i >= 0 {
		assert.LessOrEqual(t, len(value)-i-1, 2, fmt.Sprintf(format, args...)+": %q has more than two decimals", value)
	}
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// TestDocumentInvariants converts a wide spread of document shapes and checks
// every arithmetic identity on each. It is the guard on the totals
// restatement: a shape that cannot be restated has to fall back cleanly rather
// than emit a document that contradicts itself.
func TestDocumentInvariants(t *testing.T) {
	std := func(price string) *bill.Line { return vatLine(t, price, standardVAT("15%")) }
	reduced := func(price string) *bill.Line { return vatLine(t, price, standardVAT("5%")) }
	zeroRated := func(price string) *bill.Line {
		return vatLine(t, price, untaxedVAT(tax.KeyZero, "Z", "VATEX-SA-35"))
	}
	exempt := func(price string) *bill.Line {
		return vatLine(t, price, untaxedVAT(tax.KeyExempt, "E", "VATEX-SA-29"))
	}
	outside := func(price string) *bill.Line {
		return vatLine(t, price, untaxedVAT(cbc.KeyEmpty, "O", "VATEX-SA-OOS"))
	}

	// The prices the cent hides in: each divides by 1.15 into a net that sits
	// just the wrong side of a half-cent.
	awkward := []string{"1002.30", "10.00", "333.33", "0.05", "999.99"}
	awkwardLines := func() []*bill.Line {
		out := make([]*bill.Line, 0, len(awkward))
		for _, p := range awkward {
			out = append(out, std(p))
		}
		return out
	}

	specs := []invoiceSpec{
		{name: "standard rate, prices exclusive", lines: []*bill.Line{std("100.00"), std("33.33")}},
		{name: "standard rate, prices inclusive", pricesInclude: true, lines: awkwardLines()},
		{name: "simplified, prices inclusive", simplified: true, pricesInclude: true, lines: awkwardLines()},
		{name: "zero rated only", lines: []*bill.Line{zeroRated("50.00"), zeroRated("21.74")}},
		{name: "exempt only", lines: []*bill.Line{exempt("200.00")}},
		{name: "outside scope only", lines: []*bill.Line{outside("300.00")}},
		{name: "every category mixed", lines: []*bill.Line{std("1002.30"), zeroRated("50.00"), exempt("200.00"), outside("300.00")}},
		{name: "every category mixed, prices inclusive", pricesInclude: true,
			lines: []*bill.Line{std("1002.30"), zeroRated("50.00"), exempt("200.00"), outside("300.00")}},
		{name: "two standard rates", lines: []*bill.Line{std("1002.30"), reduced("1002.30")}},
		{name: "two standard rates, prices inclusive", pricesInclude: true,
			lines: []*bill.Line{std("1002.30"), reduced("1002.30")}},
		{name: "credit note", docType: bill.InvoiceTypeCreditNote, pricesInclude: true, lines: awkwardLines()},
		{name: "debit note", docType: bill.InvoiceTypeDebitNote, pricesInclude: true, lines: awkwardLines()},
		{name: "zero amount line", lines: []*bill.Line{std("0.00"), std("1002.30")}},
		{name: "single awkward line", pricesInclude: true, lines: []*bill.Line{std("1002.30")}},
	}

	// A line discount rides inside the line total, so the line VAT still
	// accounts for it. ZATCA forbids line charges (BR-KSA-EN16931-06).
	discounted := func() *bill.Line {
		l := std("1002.30")
		l.Discounts = []*bill.LineDiscount{{Reason: "Loyalty", Amount: amt(t, "2.30")}}
		return l
	}
	specs = append(specs,
		invoiceSpec{name: "line discount", lines: []*bill.Line{discounted(), std("10.00")}},
		invoiceSpec{name: "line discount, prices inclusive", pricesInclude: true,
			lines: []*bill.Line{discounted(), std("10.00")}},
	)

	// Document-level allowances and charges carry VAT of their own, which no
	// line accounts for. This is the shape the restatement has to leave alone.
	// Fresh values per spec: Calculate divides a tax-inclusive amount in place,
	// so a shared charge would arrive already divided at the next scenario.
	docCharge := func() []*bill.Charge {
		return []*bill.Charge{{
			Reason: "Freight", Amount: amt(t, "11.00"),
			Taxes: tax.Set{standardVAT("15%")},
			Ext:   tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyCharge: "ABK"}),
		}}
	}
	docDiscount := func() []*bill.Discount {
		return []*bill.Discount{{
			Reason: "Promotion", Amount: amt(t, "10.00"),
			Taxes: tax.Set{standardVAT("15%")},
			Ext:   tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyAllowance: "88"}),
		}}
	}
	specs = append(specs,
		invoiceSpec{name: "document charge", lines: awkwardLines(), charges: docCharge()},
		invoiceSpec{name: "document discount", lines: awkwardLines(), discounts: docDiscount()},
		invoiceSpec{name: "document charge, prices inclusive", pricesInclude: true, lines: awkwardLines(), charges: docCharge()},
		invoiceSpec{name: "document charge and discount, prices inclusive", pricesInclude: true,
			lines: awkwardLines(), charges: docCharge(), discounts: docDiscount()},
		invoiceSpec{name: "document discount on a zero-rated line", lines: []*bill.Line{zeroRated("50.00"), std("1002.30")},
			discounts: docDiscount()},
	)

	// A foreign-currency document states its tax twice, the second converted
	// into the accounting currency (BT-111).
	specs = append(specs, invoiceSpec{
		name: "foreign currency", currency: currency.USD, pricesInclude: true,
		lines: awkwardLines(),
		rates: []*currency.ExchangeRate{{From: currency.USD, To: currency.SAR, Amount: amt(t, "3.75")}},
	})

	for _, spec := range specs {
		t.Run(spec.name, func(t *testing.T) {
			inv := spec.build(t)
			// Snapshot before Calculate, which divides tax-inclusive amounts
			// in place.
			charged := make([]num.Amount, len(inv.Lines))
			totalCharged := num.MakeAmount(0, 2)
			for i, l := range inv.Lines {
				charged[i] = l.Item.Price.Multiply(l.Quantity)
				totalCharged = totalCharged.Add(charged[i])
			}
			for _, c := range inv.Charges {
				totalCharged = totalCharged.Add(c.Amount)
			}
			for _, d := range inv.Discounts {
				totalCharged = totalCharged.Subtract(d.Amount)
			}

			env, err := gobl.Envelop(inv)
			require.NoError(t, err)
			doc, err := zatca.ConvertInvoice(env)
			require.NoError(t, err)

			built, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)

			// Only a plain line, with no adjustments of its own, charges the
			// price it quotes; the rest are checked by the identities alone.
			var gross []num.Amount
			if spec.pricesInclude && !hasLineAdjustments(inv) {
				gross = charged
			}
			assertInvariants(t, doc, built, gross)

			// Whatever the totals say, the customer must be billed exactly what
			// the document charged.
			if spec.pricesInclude && !hasLineAdjustments(inv) {
				assert.Equal(t, 0, amt(t, doc.LegalMonetaryTotal.PayableAmount.Value).Compare(totalCharged),
					"BT-115 (%s) must be everything the document charged (%s)",
					doc.LegalMonetaryTotal.PayableAmount.Value, totalCharged)
			}
		})
	}
}

func hasLineAdjustments(inv *bill.Invoice) bool {
	for _, l := range inv.Lines {
		if len(l.Discounts) > 0 || len(l.Charges) > 0 {
			return true
		}
	}
	return false
}
