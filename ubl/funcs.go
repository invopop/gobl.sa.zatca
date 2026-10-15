package ubl

import (
	"encoding/xml"
	"strings"

	"github.com/invopop/gobl"
	zatca "github.com/invopop/gobl.sa.zatca/addon"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
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

// exportParty gives the VAT number without its country prefix, and the
// address in the ZATCA layout.
func exportParty(p *org.Party, out *goblubl.Party) {
	exportPartyTaxScheme(p, out)
	if len(p.Addresses) > 0 && out.PostalAddress != nil {
		exportAddress(p.Addresses[0], out.PostalAddress)
	}
}

// exportPartyTaxScheme drops the country prefix GOBL keeps on a tax identity:
// the VAT registration number (BT-31) ZATCA expects is the 15 digits alone.
func exportPartyTaxScheme(p *org.Party, out *goblubl.Party) {
	if p.TaxID == nil || p.TaxID.Country == "" {
		return
	}
	prefix := p.TaxID.Country.String()
	for i := range out.PartyTaxScheme {
		if id := out.PartyTaxScheme[i].CompanyID; id != nil {
			id.Value = strings.TrimPrefix(id.Value, prefix)
		}
	}
}

// exportAddress states the address the way ZATCA reads it: the building number
// on its own (BT-KSA-08) and the district in CitySubdivisionName (BT-KSA-09),
// where EN 16931 has only a second street line.
func exportAddress(a *org.Address, out *goblubl.PostalAddress) {
	l := a.LineTwo()
	out.CitySubdivisionName = &l
	out.AdditionalStreetName = nil
	out.BuildingNumber = &a.Number
}

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

// exportPayment moves the payment details into their ZATCA positions.
func exportPayment(inv *bill.Invoice, out *goblubl.Invoice) {
	if len(out.PaymentMeans) == 0 {
		return
	}
	pm := &out.PaymentMeans[0]

	// A credit note converts as a UBL Invoice here, which states the due date
	// at the top (BT-9) rather than on the payment means as a CreditNote must.
	if pm.PaymentDueDate != nil {
		out.DueDate = *pm.PaymentDueDate
		pm.PaymentDueDate = nil
	}

	// BR-KSA-17: debit and credit notes must contain the reason they were
	// issued.
	for _, ref := range inv.Preceding {
		pm.InstructionNote = append(pm.InstructionNote, ref.Reason)
	}
}

// exportOrdering drops the sales order reference (BT-14), which ZATCA has no
// field for.
func exportOrdering(inv *bill.Invoice, out *goblubl.Invoice) {
	if out.OrderReference == nil {
		return
	}
	out.OrderReference.SalesOrderID = ""
	o := inv.Ordering
	if (o == nil || len(o.Purchases) == 0) && out.BuyerReference != "" {
		out.OrderReference = nil
	}
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

// importPartyIdentities restores the ZATCA identity types. A party identifier
// is named by its own scheme here — CRN, MOM, MLS, SAG or OTH — where
// EN 16931 expects an ISO 6523 code, so the base import files it as one.
func importPartyIdentities(p *org.Party) {
	for _, id := range p.Identities {
		if id == nil || id.Scope != "" {
			continue
		}
		scheme := id.Ext.Get(iso.ExtKeySchemeID)
		if scheme == cbc.CodeEmpty {
			continue
		}
		id.Type = scheme
		id.Ext = id.Ext.Delete(iso.ExtKeySchemeID)
	}
}

func importInvoice(in *goblubl.Invoice, out *bill.Invoice) error {
	// The base import never reads cbc:UUID (BR-KSA-03), so without this GOBL
	// mints a fresh one on every Calculate and the document loses its identity.
	if u, err := uuid.Parse(in.UUID); err == nil {
		out.UUID = u
	}

	typeCode := in.InvoiceTypeCode
	if typeCode == nil {
		typeCode = in.CreditNoteTypeCode
	}
	if typeCode != nil && typeCode.Name != nil {
		code := cbc.Code(*typeCode.Name)
		out.Tax.Ext = out.Tax.Ext.Set(zatca.ExtKeyInvoiceType, code)
		if tags := invoiceTypeTags(code); len(tags) != 0 {
			// Merge rather than replace: the base import tags a document for
			// bypass when it cannot reproduce the totals the sender declared,
			// and replacing the list would drop that and recalculate them away.
			out.SetTags(mergeTags(out.GetTags(), tags)...)
		}
	}

	// BR-KSA-17: the preceding document reasons are stored in the
	// PaymentMeans InstructionNote.
	if len(out.Preceding) > 0 && len(in.PaymentMeans) > 0 {
		for i, note := range in.PaymentMeans[0].InstructionNote {
			if i < len(out.Preceding) {
				out.Preceding[i].Reason = goblubl.CleanString(note)
			}
		}
	}
	return nil
}

// mergeTags adds tags to the ones already on the document, keeping each once.
func mergeTags(existing, add []cbc.Key) []cbc.Key {
	out := append([]cbc.Key{}, existing...)
	for _, tag := range add {
		if !tag.In(out...) {
			out = append(out, tag)
		}
	}
	return out
}

// invoiceTypeTags maps the KSA-2 transaction type flags to their tags.
func invoiceTypeTags(code cbc.Code) []cbc.Key {
	var tags []cbc.Key
	it := zatca.ParseInvoiceType(code)
	if it.Simplified {
		tags = append(tags, tax.TagSimplified)
	}
	if it.ThirdParty {
		tags = append(tags, zatca.TagThirdParty)
	}
	if it.Nominal {
		tags = append(tags, zatca.TagNominal)
	}
	if it.Export {
		tags = append(tags, tax.TagExport)
	}
	if it.Summary {
		tags = append(tags, zatca.TagSummary)
	}
	if it.SelfBilled {
		tags = append(tags, tax.TagSelfBilled)
	}
	return tags
}
