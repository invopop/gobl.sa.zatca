package ubl

import (
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// exportOrdering drops the sales order reference (BT-14), which ZATCA has no
// field for.
func exportOrdering(inv *bill.Invoice, out *goblubl.Invoice) {
	if out.OrderReference == nil {
		return
	}
	out.OrderReference.SalesOrderID = ""
	o := inv.Ordering
	if (o == nil || len(o.Purchases) == 0) && out.BuyerReference != "" {
		out.OrderReference = nil
	}
}
