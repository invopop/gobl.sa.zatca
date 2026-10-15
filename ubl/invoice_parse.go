package ubl

import (
	"slices"

	zatca "github.com/invopop/gobl.sa.zatca/addon"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
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

	if typeCode := documentTypeCode(in); typeCode != nil {
		if typeCode.Name != nil {
			out.Tax.Ext = out.Tax.Ext.Set(zatca.ExtKeyInvoiceType, cbc.Code(*typeCode.Name))
		}
		importDocumentType(typeCode, out)
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

// documentType pairs a BT-3 code with the invoice type and tags that stand
// for it.
type documentType struct {
	typ  cbc.Key
	tags []cbc.Key
}

// documentTypes inverts the ZATCA scenarios: each code BR-KSA-05 accepts,
// against the invoice type and tags whose scenario produces that same code.
var documentTypes = map[cbc.Code]documentType{
	"388": {bill.InvoiceTypeStandard, nil},
	"386": {bill.InvoiceTypeStandard, []cbc.Key{tax.TagPrepayment}},
	"383": {bill.InvoiceTypeDebitNote, nil},
	"381": {bill.InvoiceTypeCreditNote, nil},
}

// documentTypeCode provides BT-3, with the KSA-2 flags as its name, whichever
// element the document carries it in.
func documentTypeCode(in *goblubl.Invoice) *goblubl.IDType {
	if in.InvoiceTypeCode != nil {
		return in.InvoiceTypeCode
	}
	return in.CreditNoteTypeCode
}

// importDocumentType sets the invoice type and tags from BT-3 and the KSA-2
// flags. The base import derives both from BT-3 alone, and nothing else tags
// the invoice before a format's import funcs run, so they are replaced
// outright. A code ZATCA does not accept is kept as stated on an "other"
// invoice, so BR-KSA-05 reports it instead of the scenarios turning it into a
// 388 or a 381.
func importDocumentType(typeCode *goblubl.IDType, out *bill.Invoice) {
	code := cbc.Code(typeCode.Value)
	dt, ok := documentTypes[code]
	if !ok {
		dt = documentType{typ: bill.InvoiceTypeOther}
		out.Tax.Ext = out.Tax.Ext.Set(untdid.ExtKeyDocumentType, code)
	}

	// A copy, as SetTags keeps the slice and the table is shared.
	tags := slices.Clone(dt.tags)
	if typeCode.Name != nil {
		tags = append(tags, invoiceTypeTags(cbc.Code(*typeCode.Name))...)
	}
	out.Type = dt.typ
	out.SetTags(tags...)
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
