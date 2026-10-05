package zatca

import "github.com/invopop/gobl/bill"

// applyPayment moves the payment details into their ZATCA positions.
func (ui *Invoice) applyPayment(inv *bill.Invoice) {
	if len(ui.PaymentMeans) == 0 {
		return
	}
	pm := &ui.PaymentMeans[0]

	// A credit note converts as a UBL Invoice here, which states the due date
	// at the top (BT-9) rather than on the payment means as a CreditNote must.
	if pm.PaymentDueDate != nil {
		ui.DueDate = *pm.PaymentDueDate
		pm.PaymentDueDate = nil
	}

	// BR-KSA-17: Debit and credit note must contain the
	// reason for this invoice type issuing.
	for _, ref := range inv.Preceding {
		pm.InstructionNote = append(pm.InstructionNote, ref.Reason)
	}
}
