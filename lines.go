package zatca

import (
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// grossAmounts records what each line charges before the base converter divides
// tax-inclusive prices down to their net amounts.
type grossAmounts []num.Amount

// documentGross snapshots those amounts, or nothing when the prices exclude tax
// and there is nothing to divide out.
func documentGross(inv *bill.Invoice) grossAmounts {
	if inv.Tax == nil || inv.Tax.PricesInclude.IsEmpty() {
		return nil
	}
	out := make(grossAmounts, len(inv.Lines))
	for i, l := range inv.Lines {
		if l != nil && l.Total != nil {
			out[i] = rescaleToCurrency(*l.Total, lineCurrency(inv, l))
		}
	}
	return out
}

// line returns what line i charged, or nil when the prices excluded tax.
func (g grossAmounts) line(i int) *num.Amount {
	if i >= len(g) {
		return nil
	}
	return &g[i]
}

// lineCurrency returns the currency a line's amounts are stated in.
func lineCurrency(inv *bill.Invoice, l *bill.Line) string {
	if l != nil && l.Item != nil && l.Item.Currency != currency.CodeEmpty {
		return l.Item.Currency.String()
	}
	return inv.Currency.String()
}

// applyLines states the line VAT amount (KSA-11) and the line amount with VAT
// (KSA-12), which BR-KSA-52 and BR-KSA-53 make mandatory and which EN 16931 has
// no line-level equivalent of.
func (ui *Invoice) applyLines(inv *bill.Invoice, gross grossAmounts) {
	for i := range ui.InvoiceLines {
		if i >= len(inv.Lines) || inv.Lines[i] == nil {
			break
		}
		line := &ui.InvoiceLines[i]
		ccy := lineCurrency(inv, inv.Lines[i])
		net, err := num.AmountFromString(line.LineExtensionAmount.Value)
		if err != nil {
			continue
		}
		vat := lineVATAmount(inv.Lines[i], net, gross.line(i), ccy)
		line.TaxTotal = []ubl.TaxTotal{
			{
				// KSA-11: line VAT amount
				TaxAmount: newAmount(vat, ccy),
				// KSA-12: line amount inclusive VAT
				RoundingAmount: newAmountPtr(net.Add(vat), ccy),
			},
		}
	}
}

// lineVATAmount works out a line's VAT (KSA-11).
func lineVATAmount(l *bill.Line, net num.Amount, gross *num.Amount, ccy string) num.Amount {
	if gross != nil {
		return gross.Subtract(net)
	}
	if p := lineVATPercent(l); p != nil {
		return rescaleToCurrency(p.Of(net), ccy)
	}
	return rescaleToCurrency(num.AmountZero, ccy)
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
