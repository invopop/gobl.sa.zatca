package zatca

import (
	"encoding/xml"

	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.sa.zatca/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// rootNameInvoice is the element ZATCA carries every document in, credit and
// debit notes included.
const rootNameInvoice = "Invoice"

// rootNameCreditNote is the element EN 16931 carries a credit note in.
const rootNameCreditNote = "CreditNote"

// buildInvoice builds the plain EN 16931 document, then reworks it into ZATCA.
func buildInvoice(env *gobl.Envelope) (*Invoice, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, ErrUnsupportedDocumentType
	}
	if err := ensureAddon(env, inv); err != nil {
		return nil, err
	}
	// gobl.ubl divides tax-inclusive prices down to their net amounts inside its
	// own Convert below, so what each line charges is read off here.
	gross := documentGross(inv)

	base, err := ubl.ConvertInvoice(env, ubl.WithContext(ublContext))
	if err != nil {
		return nil, err
	}
	out := (*Invoice)(base)
	out.applyZATCA(inv, gross)
	return out, nil
}

// ensureAddon adds the addon and recalculates, so its normalizations run.
func ensureAddon(env *gobl.Envelope, inv *bill.Invoice) error {
	if !addon.V1.In(inv.GetAddons()...) {
		inv.SetAddons(append(inv.GetAddons(), addon.V1)...)
		if err := env.Calculate(); err != nil {
			return err
		}
	}
	return env.Validate()
}

// applyZATCA reworks the EN 16931 document into ZATCA Phase 2.
func (ui *Invoice) applyZATCA(inv *bill.Invoice, gross grossAmounts) {
	ui.applyInvoiceRoot()
	ui.applyHeader(inv)
	ui.applyParties(inv)
	ui.applyOrdering(inv)
	ui.applyPayment(inv)
	ui.applyLines(inv, gross)
	ui.applyTotals(inv)
}

// applyInvoiceRoot moves a credit or debit note back into a UBL Invoice document
func (ui *Invoice) applyInvoiceRoot() {
	if ui.XMLName.Local != rootNameCreditNote {
		return
	}
	ui.XMLName = xml.Name{Local: rootNameInvoice}
	ui.UBLNamespace = ubl.NamespaceUBLInvoice
	ui.InvoiceTypeCode, ui.CreditNoteTypeCode = ui.CreditNoteTypeCode, nil
	ui.InvoiceLines, ui.CreditNoteLines = ui.CreditNoteLines, nil
	for i := range ui.InvoiceLines {
		l := &ui.InvoiceLines[i]
		l.InvoicedQuantity, l.CreditedQuantity = l.CreditedQuantity, nil
	}
}

// applyHeader sets the document-level fields ZATCA adds to EN 16931.
func (ui *Invoice) applyHeader(inv *bill.Invoice) {
	ui.SchemaLocation = ""
	// BR-KSA-EN16931-01
	ui.CustomizationID = CustomizationID
	ui.ProfileID = &ubl.IDType{Value: ProfileID}
	// BR-KSA-03
	ui.UUID = string(inv.UUID)
	// BR-KSA-70
	ui.IssueTime = inv.IssueTime.String()
	// BR-KSA-70
	ui.TaxCurrencyCode = string(inv.RegimeDef().GetCurrency())
	// BR-KSA-06
	invType := inv.Tax.GetExt(addon.ExtKeyInvoiceType).String()
	ui.InvoiceTypeCode.Name = &invType
}
