package ubl

import (
	"encoding/xml"

	zatca "github.com/invopop/gobl.sa.zatca/addon"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// LayerZATCA applies the ZATCA rules on top of the base conversion.
var LayerZATCA = &goblubl.Layer{
	ConvertInvoice: convertInvoice,
	ConvertParty: func(_ *goblubl.Context, p *org.Party, out *goblubl.Party) {
		// The VAT number is given without its country prefix.
		if p.TaxID != nil && p.TaxID.Code != "" && len(out.PartyTaxScheme) > 0 {
			id := out.PartyTaxScheme[0].CompanyID
			id.Value = id.Value[2:]
		}
		if len(p.Addresses) > 0 && out.PostalAddress != nil {
			convertAddress(p.Addresses[0], out.PostalAddress)
		}
	},
	ParseParty: func(_ *goblubl.Context, in *goblubl.Party, out *org.Party) {
		// ZATCA identities are described by their type, without an ISO
		// scheme extension. They are the last identities parsed.
		ids := make([]*goblubl.IDType, 0, len(in.PartyIdentification))
		for _, pi := range in.PartyIdentification {
			if pi.ID != nil {
				ids = append(ids, pi.ID)
			}
		}
		parsed := out.Identities[len(out.Identities)-len(ids):]
		for i, id := range ids {
			if id.SchemeID != nil {
				parsed[i].Type = cbc.Code(*id.SchemeID)
				parsed[i].Ext = tax.Extensions{}
			}
		}
	},
	ParseInvoice: parseInvoice,
}

func convertInvoice(_ *goblubl.Context, inv *bill.Invoice, out *goblubl.Invoice) error {
	// ZATCA treats all documents as invoices.
	if out.CreditNoteTypeCode != nil {
		creditNoteAsInvoice(inv, out)
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

	// BR-KSA-EN16931-09: the tax total is repeated in the accounting
	// currency even when it matches the document currency.
	if inv.Totals != nil && (taxCurrency == "" || inv.Currency == taxCurrency) {
		out.TaxTotal = append(out.TaxTotal, goblubl.TaxTotal{
			TaxAmount: goblubl.NewAmount(inv.Totals.Tax, inv.Currency.String()),
		})
	}

	// Line VAT amount (KSA-11) is mandatory for tax invoices and the
	// associated credit and debit notes.
	for i, l := range inv.Lines {
		if l.Total == nil || len(l.Taxes) == 0 || l.Taxes[0].Percent == nil {
			continue
		}
		ccy := l.Item.Currency.String()
		if ccy == "" {
			ccy = inv.Currency.String()
		}
		taxAmount := l.Taxes[0].Percent.Of(*l.Total)
		rounding := goblubl.NewAmount(l.Total.Add(taxAmount), ccy)
		out.InvoiceLines[i].TaxTotal = []goblubl.TaxTotal{
			{
				TaxAmount:      goblubl.NewAmount(taxAmount, ccy),
				RoundingAmount: &rounding,
			},
		}
	}

	// BR-KSA-17: debit and credit notes must contain the reason they were
	// issued.
	if inv.Preceding != nil && inv.Payment != nil && inv.Payment.Instructions != nil && len(out.PaymentMeans) > 0 {
		for _, ref := range inv.Preceding {
			out.PaymentMeans[0].InstructionNote = append(out.PaymentMeans[0].InstructionNote, ref.Reason)
		}
	}

	// BT-14: the sales order reference does not apply.
	if o := inv.Ordering; o != nil && len(o.Sales) > 0 && out.OrderReference != nil {
		out.OrderReference.SalesOrderID = ""
		if len(o.Purchases) == 0 && out.BuyerReference != "" {
			out.OrderReference = nil
		}
	}

	if d := inv.Delivery; d != nil && d.Receiver != nil && len(d.Receiver.Addresses) > 0 && len(out.Delivery) > 0 {
		if loc := out.Delivery[0].DeliveryLocation; loc != nil && loc.Address != nil {
			convertAddress(d.Receiver.Addresses[0], loc.Address)
		}
	}
	return nil
}

// creditNoteAsInvoice restores the invoice structure that the base
// conversion replaced for a credit note.
func creditNoteAsInvoice(inv *bill.Invoice, out *goblubl.Invoice) {
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

	// Credit notes carry the due date in the payment means.
	if len(out.PaymentMeans) > 0 {
		out.PaymentMeans[0].PaymentDueDate = nil
	}
	if p := inv.Payment; p != nil && p.Terms != nil && len(p.Terms.DueDates) > 0 {
		out.DueDate = goblubl.FormatDate(*p.Terms.DueDates[0].Date)
	}
}

func convertAddress(a *org.Address, out *goblubl.PostalAddress) {
	l := a.LineTwo()
	out.CitySubdivisionName = &l
	out.AdditionalStreetName = nil
	out.BuildingNumber = &a.Number
}

func parseInvoice(_ *goblubl.Context, in *goblubl.Invoice, out *bill.Invoice) error {
	if tc := in.InvoiceTypeCode; tc != nil && tc.Name != nil {
		out.Tax.Ext = out.Tax.Ext.Set(zatca.ExtKeyInvoiceType, cbc.Code(*tc.Name))
	}

	typeCode := in.InvoiceTypeCode
	if typeCode == nil {
		typeCode = in.CreditNoteTypeCode
	}
	if typeCode != nil && typeCode.Name != nil {
		out.Tags = tax.Tags{}
		if tags := invoiceTypeTags(cbc.Code(*typeCode.Name)); len(tags) != 0 {
			out.SetTags(tags...)
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
