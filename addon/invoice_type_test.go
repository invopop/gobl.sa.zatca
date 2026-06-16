package zatca

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInvoiceType(t *testing.T) {
	t.Run("standard with flags", func(t *testing.T) {
		assert.Equal(t, InvoiceType{
			Simplified: false,
			ThirdParty: true,
			Nominal:    true,
			Export:     true,
			Summary:    true,
			SelfBilled: false,
		}, ParseInvoiceType("0111110"))
	})

	t.Run("simplified summary", func(t *testing.T) {
		assert.Equal(t, InvoiceType{Simplified: true, Summary: true}, ParseInvoiceType("0200010"))
	})

	t.Run("standard self-billed", func(t *testing.T) {
		assert.Equal(t, InvoiceType{SelfBilled: true}, ParseInvoiceType("0100001"))
	})

	t.Run("absent or malformed code yields zero value", func(t *testing.T) {
		assert.Equal(t, InvoiceType{}, ParseInvoiceType(""))
		assert.Equal(t, InvoiceType{}, ParseInvoiceType("010000")) // too short
	})
}

// TestValidTransactionTypes confirms the codes are sourced from the extension
// definition (all 32) and that each parses consistently with its prefix.
func TestValidTransactionTypes(t *testing.T) {
	assert.Len(t, validTransactionTypes, 32)
	for _, code := range validTransactionTypes {
		assert.Equalf(t, code.String()[:2] == "02", ParseInvoiceType(code).Simplified,
			"code %s: parsed Simplified flag should match prefix", code)
	}
}
