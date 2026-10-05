package zatca

import (
	"fmt"
	"strings"

	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.sa.zatca/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
)

// Parse reads a ZATCA document off the wire.
func Parse(data []byte) (any, error) {
	doc, err := ubl.Parse(data)
	if err != nil {
		return nil, err
	}
	in, ok := doc.(*ubl.Invoice)
	if !ok {
		return nil, ErrUnknownDocumentType
	}
	return (*Invoice)(in), nil
}

// ParseInvoice is Parse for callers that already know it is an invoice.
func ParseInvoice(data []byte) (*Invoice, error) {
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	inv, ok := doc.(*Invoice)
	if !ok {
		return nil, fmt.Errorf("expected invoice, got %T", doc)
	}
	return inv, nil
}

// ExtractBinaryAttachments returns the files embedded in the document.
func (ui *Invoice) ExtractBinaryAttachments() []BinaryAttachment {
	return (*ubl.Invoice)(ui).ExtractBinaryAttachments()
}

// Convert runs the generic parse and then adds the ZATCA details the base has
// no field for.
func (ui *Invoice) Convert() (*gobl.Envelope, error) {
	if ui.CustomizationID != CustomizationID {
		return nil, fmt.Errorf("unsupported customization id %q, expected %q", ui.CustomizationID, CustomizationID)
	}

	env, err := (*ubl.Invoice)(ui).Convert(ubl.WithContext(ublContext))
	if err != nil {
		return nil, err
	}
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, ErrUnsupportedDocumentType
	}

	ui.addGOBLDetails(inv)

	if err := env.Calculate(); err != nil {
		return nil, err
	}
	return env, nil
}

// addGOBLDetails fills in the GOBL fields the generic parse leaves empty,
// reading them off the original ZATCA document.
func (ui *Invoice) addGOBLDetails(inv *bill.Invoice) {
	// The generic parse never reads cbc:UUID (BR-KSA-03), so without this GOBL
	// mints a fresh one on every Calculate and the document loses its identity.
	if u, err := uuid.Parse(ui.UUID); err == nil {
		inv.UUID = u
	}
	ui.addInvoiceType(inv)
	ui.addPrecedingReasons(inv)
	applyPartyIdentities(inv.Supplier)
	applyPartyIdentities(inv.Customer)
}

// addInvoiceType restores the KSA-2 transaction type, which ZATCA carries as
// the type code's name attribute, along with the GOBL tags that describe it.
func (ui *Invoice) addInvoiceType(inv *bill.Invoice) {
	if ui.InvoiceTypeCode == nil || ui.InvoiceTypeCode.Name == nil {
		return
	}
	code := cbc.Code(*ui.InvoiceTypeCode.Name)
	if inv.Tax == nil {
		inv.Tax = new(bill.Tax)
	}
	inv.Tax.Ext = inv.Tax.Ext.Set(addon.ExtKeyInvoiceType, code)
	if tags := invoiceTags(code); len(tags) > 0 {
		// Merge rather than replace: the generic parse tags a document for
		// bypass when it cannot reproduce the totals the sender declared, and
		// replacing the list would drop that and recalculate them away.
		inv.SetTags(mergeTags(inv.GetTags(), tags)...)
	}
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

// invoiceTags maps the KSA-2 transaction type flags onto GOBL tags.
func invoiceTags(code cbc.Code) []cbc.Key {
	it := addon.ParseInvoiceType(code)
	var tags []cbc.Key
	if it.Simplified {
		tags = append(tags, tax.TagSimplified)
	}
	if it.ThirdParty {
		tags = append(tags, addon.TagThirdParty)
	}
	if it.Nominal {
		tags = append(tags, addon.TagNominal)
	}
	if it.Export {
		tags = append(tags, tax.TagExport)
	}
	if it.Summary {
		tags = append(tags, addon.TagSummary)
	}
	if it.SelfBilled {
		tags = append(tags, tax.TagSelfBilled)
	}
	return tags
}

// addPrecedingReasons pairs the payment means instruction notes with the
// preceding documents by index. BR-KSA-17 puts the reason a credit or debit
// note was issued there, where GOBL keeps it on the reference itself.
func (ui *Invoice) addPrecedingReasons(inv *bill.Invoice) {
	if len(inv.Preceding) == 0 || len(ui.PaymentMeans) == 0 {
		return
	}
	for i, note := range ui.PaymentMeans[0].InstructionNote {
		if i >= len(inv.Preceding) {
			break
		}
		inv.Preceding[i].Reason = cleanString(note)
	}
}

// applyPartyIdentities restores the ZATCA identity types. A party identifier is
// named by its own scheme here — CRN, MOM, MLS, SAG or OTH — where EN 16931
// expects an ISO 6523 code, so the base parse files it as one.
func applyPartyIdentities(p *org.Party) {
	if p == nil {
		return
	}
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

// cleanString drops the replacement characters a mis-encoded document arrives
// with, as the base parse does for the text it reads itself.
func cleanString(s string) string {
	return strings.ReplaceAll(s, "�", "")
}
