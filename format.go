package zatca

import (
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
)

// newAmount builds a monetary amount at the natural precision of the given
// currency, which BR-KSA-DEC-01 through BR-KSA-DEC-05 cap every ZATCA amount at.
func newAmount(a num.Amount, ccy string) ubl.Amount {
	value := rescaleToCurrency(a, ccy).String()
	return ubl.Amount{CurrencyID: &ccy, Value: value}
}

// newAmountPtr is newAmount for the optional elements.
func newAmountPtr(a num.Amount, ccy string) *ubl.Amount {
	amount := newAmount(a, ccy)
	return &amount
}

// rescaleToCurrency rounds an amount to the natural precision of the given
// currency, leaving it alone when the currency is unknown.
func rescaleToCurrency(a num.Amount, ccy string) num.Amount {
	if def := currency.Code(ccy).Def(); def != nil {
		return def.Rescale(a)
	}
	return a
}

// ptr addresses a string, which most optional UBL fields need.
func ptr(s string) *string {
	return &s
}
