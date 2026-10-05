package zatca

import (
	"strings"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
)

// applyParties reworks every party the base built into its ZATCA form.
func (ui *Invoice) applyParties(inv *bill.Invoice) {
	applyParty(ui.AccountingSupplierParty.Party, inv.Supplier)
	applyParty(ui.AccountingCustomerParty.Party, inv.Customer)
	if inv.Payment != nil {
		applyParty(ui.PayeeParty, inv.Payment.Payee)
	}
	if inv.Ordering != nil {
		applyParty(ui.TaxRepresentativeParty, inv.Ordering.Seller)
	}
}

func applyParty(p *ubl.Party, party *org.Party) {
	if p == nil || party == nil {
		return
	}
	applyPartyTaxScheme(p, party)
	applyPartyAddress(p, party)
}

// applyPartyTaxScheme drops the country prefix GOBL keeps on a tax identity:
// the VAT registration number (BT-31) ZATCA expects is the 15 digits alone.
func applyPartyTaxScheme(p *ubl.Party, party *org.Party) {
	if party.TaxID == nil || party.TaxID.Country == "" {
		return
	}
	prefix := party.TaxID.Country.String()
	for i := range p.PartyTaxScheme {
		if id := p.PartyTaxScheme[i].CompanyID; id != nil {
			id.Value = strings.TrimPrefix(id.Value, prefix)
		}
	}
}

// applyPartyAddress states the address the way ZATCA reads it: the building
// number on its own (BT-KSA-08) and the district in CitySubdivisionName
// (BT-KSA-09), where EN 16931 has only a second street line.
func applyPartyAddress(p *ubl.Party, party *org.Party) {
	addr := p.PostalAddress
	if addr == nil || len(party.Addresses) == 0 {
		return
	}
	a := party.Addresses[0]
	addr.BuildingNumber = ptr(a.Number)
	addr.CitySubdivisionName = ptr(a.LineTwo())
	addr.AdditionalStreetName = nil
}
