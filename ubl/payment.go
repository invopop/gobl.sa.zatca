package ubl

import (
	goblubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// exportPayment moves the payment details into their ZATCA positions.
func exportPayment(inv *bill.Invoice, out *goblubl.Invoice) {
	if len(out.PaymentMeans) == 0 {
		return
	}
	pm := &out.PaymentMeans[0]

	// A credit note converts as a UBL Invoice here, which states the due date
	// at the top (BT-9) rather than on the payment means as a CreditNote must.
	if pm.PaymentDueDate != nil {
		out.DueDate = *pm.PaymentDueDate
		pm.PaymentDueDate = nil
	}

	// BR-KSA-17: debit and credit notes must contain the reason they were
	// issued.
	for _, ref := range inv.Preceding {
		pm.InstructionNote = append(pm.InstructionNote, ref.Reason)
	}
}
