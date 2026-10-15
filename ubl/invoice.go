package ubl

import (
	"encoding/xml"

	zatca "github.com/invopop/gobl.sa.zatca/addon"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

func exportInvoice(inv *bill.Invoice, out *goblubl.Invoice) error {
	// ZATCA treats all documents as invoices.
	if out.CreditNoteTypeCode != nil {
		creditNoteAsInvoice(out)
	}

	out.SchemaLocation = ""
	// BR-KSA-03
	out.UUID = string(inv.UUID)
	// BR-KSA-70
	out.IssueTime = inv.IssueTime.String()
	// BR-KSA-70
	taxCurrency := inv.RegimeDef().GetCurrency()
	out.TaxCurrencyCode = string(taxCurrency)
	// BR-KSA-06
	invType := inv.Tax.GetExt(zatca.ExtKeyInvoiceType).String()
	out.InvoiceTypeCode.Name = &invType

	exportTotals(inv, out, taxCurrency.String())
	exportLines(inv, out)
	exportPayment(inv, out)
	exportOrdering(inv, out)

	if d := inv.Delivery; d != nil && d.Receiver != nil && len(d.Receiver.Addresses) > 0 && len(out.Delivery) > 0 {
		if loc := out.Delivery[0].DeliveryLocation; loc != nil && loc.Address != nil {
			exportAddress(d.Receiver.Addresses[0], loc.Address)
		}
	}
	return nil
}

// creditNoteAsInvoice restores the invoice structure that the base
// conversion replaced for a credit note.
func creditNoteAsInvoice(out *goblubl.Invoice) {
	out.XMLName = xml.Name{Local: "Invoice"}
	out.UBLNamespace = goblubl.NamespaceUBLInvoice
	out.InvoiceTypeCode = out.CreditNoteTypeCode
	out.CreditNoteTypeCode = nil

	out.InvoiceLines = out.CreditNoteLines
	out.CreditNoteLines = nil
	for i := range out.InvoiceLines {
		l := &out.InvoiceLines[i]
		l.InvoicedQuantity = l.CreditedQuantity
		l.CreditedQuantity = nil
	}
}
