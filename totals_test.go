package zatca_test

import (
	"testing"

	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTotalsLeftAsCalculated pins the one place a ZATCA document does not add
// up, on the invoice the customer reported. The lines state the VAT each price
// actually contains, 305.95 in total; GOBL derives the document VAT by applying
// the rate to the summed base and gets 305.96, carrying the cent in the payable
// rounding amount (BT-114).
//
// The document is left that way on purpose. No ZATCA rule compares the line VAT
// amounts to BT-110 - a document stating a hundred riyals of difference passes
// the schematron - and restating the totals here would make the XML disagree
// with the GOBL envelope that the PDF and everything else is rendered from.
func TestTotalsLeftAsCalculated(t *testing.T) {
	doc := lineTaxTest(t, true,
		standardLine("1002.30"),
		standardLine("10.00"),
		standardLine("333.33"),
		standardLine("0.05"),
		standardLine("999.99"),
	)

	sum := num.MakeAmount(0, 2)
	for _, line := range doc.InvoiceLines {
		require.Len(t, line.TaxTotal, 1)
		vat, err := num.AmountFromString(line.TaxTotal[0].TaxAmount.Value)
		require.NoError(t, err)
		sum = sum.Add(vat)
	}
	assert.Equal(t, "305.95", sum.String(), "the VAT the lines state (KSA-11)")

	require.Len(t, doc.TaxTotal, 2)
	assert.Equal(t, "305.96", doc.TaxTotal[0].TaxAmount.Value, "BT-110, as GOBL calculated it")
	assert.Equal(t, "305.96", doc.TaxTotal[1].TaxAmount.Value, "the BR-KSA-EN16931-09 repeat")

	mt := doc.LegalMonetaryTotal
	assert.Equal(t, "2039.72", mt.TaxExclusiveAmount.Value, "BT-109")
	assert.Equal(t, "2345.68", mt.TaxInclusiveAmount.Value, "BT-112 = BT-109 + BT-110 (BR-CO-15)")
	require.NotNil(t, mt.PayableRoundingAmount)
	assert.Equal(t, "-0.01", mt.PayableRoundingAmount.Value, "BT-114 carries the cent")
	require.NotNil(t, mt.PayableAmount)
	assert.Equal(t, "2345.67", mt.PayableAmount.Value, "what the customer owes is exactly what was charged")
}
