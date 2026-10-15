package ubl

import (
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// workingAccuracy is the number of decimal places added to a unit price before
// the VAT rate is applied to it, so that the rate is not applied to an amount
// already rounded.
const workingAccuracy uint32 = 2

// exportLines states the line VAT amount (KSA-11) and the line amount with VAT
// (KSA-12), which BR-KSA-52 and BR-KSA-53 make mandatory and which EN 16931 has
// no line-level equivalent of.
func exportLines(inv *bill.Invoice, out *goblubl.Invoice) {
	for i := range out.InvoiceLines {
		if i >= len(inv.Lines) || inv.Lines[i] == nil {
			break
		}
		line := &out.InvoiceLines[i]
		ccy := lineCurrency(inv, inv.Lines[i])
		net, err := num.AmountFromString(line.LineExtensionAmount.Value)
		if err != nil {
			continue
		}
		charged := lineCharged(inv.Lines[i], net, ccy)
		line.TaxTotal = []goblubl.TaxTotal{
			{
				// KSA-11: line VAT amount
				TaxAmount: goblubl.NewAmount(charged.Subtract(net), ccy),
				// KSA-12: line amount inclusive VAT
				RoundingAmount: newAmountPtr(charged, ccy),
			},
		}
	}
}

// lineCharged works out what a line charges including VAT (KSA-12), which
// KSA-11 is then the difference to. Applying the rate to the rounded net
// amount is a cent out whenever the price included the tax, so the amount is
// rebuilt from the unit price where that accuracy survives.
func lineCharged(l *bill.Line, net num.Amount, ccy string) num.Amount {
	p := lineVATPercent(l)
	if p == nil {
		// Exempt, zero-rated and out-of-scope lines carry no rate, and
		// BR-KSA-52 wants a zero amount stated rather than the line omitting
		// it altogether.
		return net
	}
	base := net
	if t := preciseLineTotal(l, ccy); t != nil {
		base = *t
	}
	return rescaleToCurrency(base.Add(p.Of(base)), ccy)
}

// preciseLineTotal rebuilds a line total from a unit price holding more
// decimal places than the currency, which is how the amount a tax-inclusive
// price charged survives being divided down to a net. It returns nil when the
// price carries no more accuracy than the net already states.
func preciseLineTotal(l *bill.Line, ccy string) *num.Amount {
	if l.Item == nil || l.Item.Price == nil {
		return nil
	}
	def := currency.Code(ccy).Def()
	if def == nil || l.Item.Price.Exp() <= def.Subunits {
		return nil
	}
	total := l.Item.Price.Upscale(workingAccuracy).Multiply(l.Quantity)
	for _, d := range l.Discounts {
		if d != nil {
			total = total.Subtract(d.Amount)
		}
	}
	for _, c := range l.Charges {
		if c != nil {
			total = total.Add(c.Amount)
		}
	}
	return &total
}

// lineVATPercent returns the line's VAT rate, or nil when it has none.
func lineVATPercent(l *bill.Line) *num.Percentage {
	for _, combo := range l.Taxes {
		if combo != nil && combo.Category == tax.CategoryVAT {
			return combo.Percent
		}
	}
	return nil
}

// lineCurrency returns the currency a line's amounts are stated in.
func lineCurrency(inv *bill.Invoice, l *bill.Line) string {
	if l != nil && l.Item != nil && l.Item.Currency != currency.CodeEmpty {
		return l.Item.Currency.String()
	}
	return inv.Currency.String()
}

// newAmountPtr is goblubl.NewAmount for the optional elements.
func newAmountPtr(a num.Amount, ccy string) *goblubl.Amount {
	amount := goblubl.NewAmount(a, ccy)
	return &amount
}

// rescaleToCurrency rounds an amount to the natural precision of the given
// currency, which BR-KSA-DEC-01 through BR-KSA-DEC-05 cap every ZATCA amount
// at, leaving it alone when the currency is unknown.
func rescaleToCurrency(a num.Amount, ccy string) num.Amount {
	if def := currency.Code(ccy).Def(); def != nil {
		return def.Rescale(a)
	}
	return a
}
