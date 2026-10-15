package zatca

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// scenarios restate the EN 16931 UNTDID document types that ZATCA does not
// accept (BR-KSA-05 allows only 388, 386, 383 and 381). They run after the
// EN 16931 scenarios, and the last match wins:
//
//   - a standard invoice is a 388 tax invoice, whatever its other tags, as
//     KSA-2 carries self-billing and ZATCA has no partial or factored invoice;
//   - a prepayment invoice is a 386, which the 388 would otherwise replace;
//   - a self-billed credit note is a 381, as KSA-2 carries self-billing;
//   - a factored credit note, self-billed or not, stays a 396 for BR-KSA-05
//     to report, as ZATCA has no code for it.
var scenarios = []*tax.ScenarioSet{
	{
		Schema: bill.ShortSchemaInvoice,
		List: []*tax.Scenario{
			{
				Types: []cbc.Key{
					bill.InvoiceTypeStandard,
				},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: "388",
				}),
			},
			{
				Types: []cbc.Key{
					bill.InvoiceTypeStandard,
				},
				Tags: []cbc.Key{
					tax.TagPrepayment,
				},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: "386",
				}),
			},
			{
				Types: []cbc.Key{
					bill.InvoiceTypeCreditNote,
				},
				Tags: []cbc.Key{
					tax.TagSelfBilled,
				},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: "381",
				}),
			},
			{
				Types: []cbc.Key{
					bill.InvoiceTypeCreditNote,
				},
				Tags: []cbc.Key{
					tax.TagFactoring,
				},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: "396",
				}),
			},
		},
	},
}
