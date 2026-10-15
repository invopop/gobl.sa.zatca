package ubl

import (
	"strings"

	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/org"
)

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
