package zatca

import (
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/org"
)

// The UBL types a ZATCA document is built from, re-exported so an application
// handling one does not have to import the generic library alongside this.
type (
	// Amount is a monetary amount with its currency.
	Amount = ubl.Amount
	// BinaryAttachment is a file embedded in the document.
	BinaryAttachment = ubl.BinaryAttachment
	// Extension is a UBL extension, which carries the cryptographic stamp.
	Extension = ubl.Extension
	// MonetaryTotal holds the document totals (BG-22).
	MonetaryTotal = ubl.MonetaryTotal
	// Party is a supplier, customer or other party on the document.
	Party = ubl.Party
	// SignatureInformation identifies the signature inside the extension.
	SignatureInformation = ubl.SignatureInformation
	// SupplierParty wraps the supplier (BG-4).
	SupplierParty = ubl.SupplierParty
	// CustomerParty wraps the customer (BG-7).
	CustomerParty = ubl.CustomerParty
	// TaxTotal is a tax total, at document or line level.
	TaxTotal = ubl.TaxTotal
	// IDType is an identifier with its optional scheme attributes.
	IDType = ubl.IDType
)

// The UBL signature identifiers ZATCA's cryptographic stamp (KSA-15) rides on.
const (
	SignatureMethod        = ubl.SignatureMethod
	SignatureInformationID = ubl.SignatureInformationID
	ReferenceSignatureID   = ubl.ReferenceSignatureID

	NamespaceSIG = ubl.NamespaceSIG
	NamespaceSAC = ubl.NamespaceSAC
	NamespaceSBC = ubl.NamespaceSBC
)

// NewExtension builds the UBL extension a signature is carried in.
func NewExtension() *Extension {
	return ubl.NewExtension()
}

// AddAttachments adds the document references ZATCA stamps onto an invoice,
// such as the invoice counter value (KSA-16).
func (ui *Invoice) AddAttachments(attachments []*org.Attachment) {
	(*ubl.Invoice)(ui).AddAttachments(attachments)
}

// AddBinaryAttachment embeds a file in the document, as the QR code (KSA-14)
// and the previous invoice hash (KSA-13) are.
func (ui *Invoice) AddBinaryAttachment(attachment BinaryAttachment) {
	(*ubl.Invoice)(ui).AddBinaryAttachment(attachment)
}

// AddExtension adds a UBL extension to the document.
func (ui *Invoice) AddExtension(extension *Extension) {
	(*ubl.Invoice)(ui).AddExtension(extension)
}

// AddSignatureReference points the document at the signature carried in its
// extensions, which BR-KSA-60 requires of a simplified invoice.
func (ui *Invoice) AddSignatureReference(signatureMethod, referenceSignatureID string) {
	(*ubl.Invoice)(ui).AddSignatureReference(signatureMethod, referenceSignatureID)
}
