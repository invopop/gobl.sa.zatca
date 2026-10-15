package ubl

import (
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

// importPartyIdentities restores the ZATCA identity types. A party identifier
// is named by its own scheme here — CRN, MOM, MLS, SAG or OTH — where
// EN 16931 expects an ISO 6523 code, so the base import files it as one.
func importPartyIdentities(p *org.Party) {
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
