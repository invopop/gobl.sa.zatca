// Package zatca is the root of the Saudi Arabia ZATCA module for GOBL. It
// converts GOBL documents to and from the ZATCA Phase 2 profile of UBL 2.1.
package zatca

import (
	"fmt"

	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.sa.zatca/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
)

// Invoice is the ZATCA view of a UBL invoice: a defined type over ubl.Invoice,
// not an alias, so ZATCA's own methods attach to it without inheriting
// gobl.ubl's generic Convert.
type Invoice ubl.Invoice

// Version is the UBL version the generated documents declare.
const Version = ubl.Version

// ZATCA identifies its Phase 2 profile with these two values (BT-24 and BT-23).
const (
	CustomizationID = "urn:cen.eu:en16931:2017#compliant#urn:zatca.gov.sa:e-invoicing:1.0"
	ProfileID       = "reporting:1.0"
)

// VESID names the schematron rule set every document is validated against.
// ZATCA ships a single one for invoices, credit notes and debit notes alike.
const VESID = "sa.zatca:ubl-invoice:2.3.8"

// Addons lists the GOBL addons a ZATCA document needs.
var Addons = []cbc.Key{addon.V1}

var (
	// ErrUnsupportedDocumentType is returned when the GOBL document cannot be
	// converted to a ZATCA document.
	ErrUnsupportedDocumentType = ubl.ErrUnsupportedDocumentType

	// ErrUnknownDocumentType is the base's error, re-exported so callers can
	// match what Parse returns without importing gobl.ubl.
	ErrUnknownDocumentType = ubl.ErrUnknownDocumentType
)

// ublContext drives the base conversion.
var ublContext = ubl.Context{
	CustomizationID: ubl.ContextEN16931.CustomizationID,
	Addons:          Addons,
	VESIDs: ubl.VESIDMapping{
		Invoice:    VESID,
		CreditNote: VESID,
	},
}

// Convert turns a GOBL envelope into a ZATCA UBL document.
func Convert(env *gobl.Envelope) (any, error) {
	out, err := buildInvoice(env)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ConvertInvoice is Convert for callers that already know the document is an
// invoice.
func ConvertInvoice(env *gobl.Envelope) (*Invoice, error) {
	doc, err := Convert(env)
	if err != nil {
		return nil, err
	}
	inv, ok := doc.(*Invoice)
	if !ok {
		return nil, fmt.Errorf("expected invoice, got %T", doc)
	}
	return inv, nil
}

// Bytes renders a converted document as indented XML.
func Bytes(in any) ([]byte, error) {
	return ubl.Bytes(in)
}

// BytesCompact renders a converted document as XML without indentation.
func BytesCompact(in any) ([]byte, error) {
	return ubl.BytesCompact(in)
}

// GetVESID names the schematron to validate a document against.
func GetVESID(_ *bill.Invoice) string {
	return VESID
}
