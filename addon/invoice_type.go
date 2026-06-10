package zatca

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
)

// InvTypeCodeLen is the length of the ZATCA invoice transaction type (KSA-2), a
// 7-character string of the form TTXNESO:
//   - TT (0-1): main type (01 = Standard, 02 = Simplified)
//   - X  (2):   third-party transaction
//   - N  (3):   nominal supply transaction
//   - E  (4):   export invoice
//   - S  (5):   summary invoice
//   - O  (6):   self-billed invoice
const InvTypeCodeLen = 7

// invTypeCodePattern is the regular expression every KSA-2 code must match.
const invTypeCodePattern = `^0[12][01]{5}$`

// invoiceType is a parsed ZATCA invoice transaction type (KSA-2, TTXNESO). Keeping the
// flags in a single typed value lets the rules run readable boolean checks
// instead of repeatedly re-fetching the code and indexing magic byte offsets.
type invoiceType struct {
	Simplified bool // false = Standard (TT 01), true = Simplified (TT 02)
	ThirdParty bool
	Nominal    bool
	Export     bool
	Summary    bool
	SelfBilled bool
}

// invTypeFlags maps each binary flag position in the KSA-2 code to its field.
// The slice order is the canonical order of the trailing flags in the code.
var invTypeFlags = []struct {
	pos int
	get func(*invoiceType) *bool
}{
	{2, func(t *invoiceType) *bool { return &t.ThirdParty }},
	{3, func(t *invoiceType) *bool { return &t.Nominal }},
	{4, func(t *invoiceType) *bool { return &t.Export }},
	{5, func(t *invoiceType) *bool { return &t.Summary }},
	{6, func(t *invoiceType) *bool { return &t.SelfBilled }},
}

// parseInvoiceType parses a KSA-2 code into its flags. It does not report
// whether the code is valid — that is enforced by the BR-KSA-06 rule against
// validTransactionTypes — so an absent or malformed code simply yields a
// zero-valued (all flags unset) invoiceType.
func parseInvoiceType(code cbc.Code) invoiceType {
	s := code.String()
	t := invoiceType{Simplified: len(s) >= 2 && s[:2] == "02"}
	for _, f := range invTypeFlags {
		if len(s) > f.pos && s[f.pos] == '1' {
			*f.get(&t) = true
		}
	}
	return t
}

// invoiceTypeOf extracts and parses the KSA-2 code carried by an invoice.
func invoiceTypeOf(val any) invoiceType {
	inv, ok := val.(*bill.Invoice)
	if !ok || inv == nil || inv.Tax == nil {
		return invoiceType{}
	}
	return parseInvoiceType(inv.Tax.GetExt(ExtKeyInvoiceTypeTransactions))
}

// Code renders the invoice transaction type back into its 7-character KSA-2 code.
func (t invoiceType) Code() cbc.Code {
	b := []byte("0100000")
	if t.Simplified {
		b[1] = '2'
	}
	for _, f := range invTypeFlags {
		if *f.get(&t) {
			b[f.pos] = '1'
		}
	}
	return cbc.Code(b)
}

// validTransactionTypes lists every valid KSA-2 code (BR-KSA-06). The invoice
// transaction type extension definition is the single source of truth.
var validTransactionTypes = func() []cbc.Code {
	def := cbc.GetKeyDefinition(ExtKeyInvoiceTypeTransactions, extensions)
	codes := make([]cbc.Code, len(def.Values))
	for i, v := range def.Values {
		codes[i] = v.Code
	}
	return codes
}()
