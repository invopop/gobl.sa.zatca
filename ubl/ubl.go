// Package ubl adds the ZATCA UBL context on top of the gobl.ubl base
// conversion, and registers it with gobl.ubl and the GOBL convert register.
// Import it for its side effects:
//
//	import _ "github.com/invopop/gobl.sa.zatca/ubl"
package ubl

import (
	zatca "github.com/invopop/gobl.sa.zatca/addon"
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/schema"
)

// KeyZATCA identifies the ZATCA context.
const KeyZATCA cbc.Key = "ubl+sa-zatca-v1"

// ContextZATCA defines the context for Saudi Arabia ZATCA Phase 2 e-invoicing.
var ContextZATCA = goblubl.Context{
	Key:             KeyZATCA,
	Name:            i18n.NewString("UBL ZATCA"),
	Countries:       []l10n.Code{l10n.SA},
	Schemas:         []schema.ID{schema.Lookup(bill.Invoice{})},
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:zatca.gov.sa:e-invoicing:1.0",
	ProfileID:       "reporting:1.0", // BT-23
	Addons:          []cbc.Key{zatca.V1},
	VESIDs: goblubl.VESIDMapping{
		Invoice:    "sa.zatca:ubl-invoice:2.3.8",
		CreditNote: "sa.zatca:ubl-invoice:2.3.8",
	},
	Layers: []*goblubl.Layer{LayerZATCA},
}

func init() {
	goblubl.RegisterContexts(ContextZATCA)
}
