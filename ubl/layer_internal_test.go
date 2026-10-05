package ubl

import (
	"testing"

	zatca "github.com/invopop/gobl.sa.zatca/addon"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

// TestTagCodeParseZATCA confirms every KSA-2 transaction-type flag is restored
// as its tag, mirroring the addon's normalizeInvoiceType (the convert direction).
func TestTagCodeParseZATCA(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []cbc.Key
	}{
		{"Standard - no flags", "0100000", nil},
		{"Simplified", "0200000", []cbc.Key{tax.TagSimplified}},
		{"Summary", "0100010", []cbc.Key{zatca.TagSummary}},
		{"Export", "0100100", []cbc.Key{tax.TagExport}},
		{"Third-party, nominal, self-billed", "0111001", []cbc.Key{zatca.TagThirdParty, zatca.TagNominal, tax.TagSelfBilled}},
		{"Simplified, export, summary", "0200110", []cbc.Key{tax.TagSimplified, tax.TagExport, zatca.TagSummary}},
		{"All flags set", "0211111", []cbc.Key{tax.TagSimplified, zatca.TagThirdParty, zatca.TagNominal, tax.TagExport, zatca.TagSummary, tax.TagSelfBilled}},
		{"Empty code - no tags", "", nil},
		{"Malformed short code - no tags", "010000", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := invoiceTypeTags(cbc.Code(tt.code))
			assert.Equal(t, tt.expected, result)
		})
	}
}
