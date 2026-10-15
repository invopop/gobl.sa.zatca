package ubl

import (
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/org"
)

// exportAddress states the address the way ZATCA reads it: the building number
// on its own (BT-KSA-08) and the district in CitySubdivisionName (BT-KSA-09),
// where EN 16931 has only a second street line.
func exportAddress(a *org.Address, out *goblubl.PostalAddress) {
	l := a.LineTwo()
	out.CitySubdivisionName = &l
	out.AdditionalStreetName = nil
	out.BuildingNumber = &a.Number
}
