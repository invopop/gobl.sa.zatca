package ubl

import (
	"github.com/invopop/gobl"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// invoices provides the GOBL and UBL invoices of an export or import, if
// both are invoices.
func invoices(env *gobl.Envelope, doc goblubl.Document) (*bill.Invoice, *goblubl.Invoice, bool) {
	inv, ok := env.Extract().(*bill.Invoice)
	out, ok2 := doc.(*goblubl.Invoice)
	return inv, out, ok && ok2
}

// exportZATCA applies the ZATCA rules to an exported invoice.
func exportZATCA(_ *goblubl.Format, env *gobl.Envelope, doc goblubl.Document) error {
	inv, out, ok := invoices(env, doc)
	if !ok {
		return nil
	}
	for _, p := range goblubl.InvoiceParties(inv, out) {
		exportParty(p.GOBL, p.UBL)
	}
	return exportInvoice(inv, out)
}

// importZATCA applies the ZATCA rules to an imported invoice.
func importZATCA(_ *goblubl.Format, doc goblubl.Document, env *gobl.Envelope) error {
	inv, in, ok := invoices(env, doc)
	if !ok {
		return nil
	}
	for _, p := range goblubl.InvoiceParties(inv, in) {
		importPartyIdentities(p.GOBL)
	}
	return importInvoice(in, inv)
}
