package zatca

import (
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// applyTotals repeats the tax total in the VAT accounting currency, which
// BR-KSA-EN16931-09 requires beside the one carrying the breakdown. A document
// billed in another currency already states it, converted, as BT-111.
//
// The document totals themselves are left as GOBL calculated them. GOBL derives
// the VAT by applying the rate to the summed base, so it can sit a cent away
// from the sum of the per-line amounts (KSA-11) the lines state; no ZATCA rule
// compares the two, and restating the document would make the XML disagree with
// the GOBL envelope every other rendering of the invoice is built from.
func (ui *Invoice) applyTotals(inv *bill.Invoice) {
	if inv.Totals == nil || len(ui.TaxTotal) == 0 {
		return
	}
	if inv.Currency != inv.RegimeDef().GetCurrency() {
		return
	}
	ui.TaxTotal = append(ui.TaxTotal, ubl.TaxTotal{
		TaxAmount: newAmount(inv.Totals.Tax, inv.Currency.String()),
	})
}
