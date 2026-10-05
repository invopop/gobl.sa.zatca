package zatca

import "github.com/invopop/gobl/bill"

// applyOrdering drops the sales order reference (BT-14), which ZATCA has no field for.
func (ui *Invoice) applyOrdering(inv *bill.Invoice) {
	if ui.OrderReference == nil {
		return
	}
	ui.OrderReference.SalesOrderID = ""
	o := inv.Ordering
	if (o == nil || len(o.Purchases) == 0) && ui.BuyerReference != "" {
		ui.OrderReference = nil
	}
}
