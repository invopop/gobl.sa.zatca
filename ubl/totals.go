package ubl

import (
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// exportTotals repeats the tax total in the VAT accounting currency, which
// BR-KSA-EN16931-09 requires beside the one carrying the breakdown. A document
// billed in another currency already states it, converted, as BT-111.
//
// The document totals themselves are left as GOBL calculated them. GOBL derives
// the VAT by applying the rate to the summed base, so it can sit a cent away
// from the sum of the per-line amounts (KSA-11) the lines state; no ZATCA rule
// compares the two, and restating the document would make the XML disagree with
// the GOBL envelope every other rendering of the invoice is built from.
func exportTotals(inv *bill.Invoice, out *goblubl.Invoice, taxCurrency string) {
	if inv.Totals == nil || len(out.TaxTotal) == 0 {
		return
	}
	if taxCurrency != "" && inv.Currency.String() != taxCurrency {
		return
	}
	out.TaxTotal = append(out.TaxTotal, goblubl.TaxTotal{
		TaxAmount: goblubl.NewAmount(inv.Totals.Tax, inv.Currency.String()),
	})
}
