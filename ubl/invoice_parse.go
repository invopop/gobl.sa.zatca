package ubl

import (
	zatca "github.com/invopop/gobl.sa.zatca/addon"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
)

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
