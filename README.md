# GOBL ➡️ Saudi Arabia ZATCA

Saudi Arabia ZATCA (Zakat, Tax and Customs Authority) e-invoicing addon for [GOBL](https://github.com/invopop/gobl).

Released under the Apache 2.0 [LICENSE](https://github.com/invopop/gobl.sa.zatca/blob/main/LICENSE), Copyright 2026 [Invopop S.L.](https://invopop.com).

[![Lint](https://github.com/invopop/gobl.sa.zatca/actions/workflows/lint.yaml/badge.svg)](https://github.com/invopop/gobl.sa.zatca/actions/workflows/lint.yaml)
[![Test Go](https://github.com/invopop/gobl.sa.zatca/actions/workflows/test.yaml/badge.svg)](https://github.com/invopop/gobl.sa.zatca/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/invopop/gobl.sa.zatca)](https://goreportcard.com/report/github.com/invopop/gobl.sa.zatca)
[![codecov](https://codecov.io/gh/invopop/gobl.sa.zatca/graph/badge.svg)](https://codecov.io/gh/invopop/gobl.sa.zatca)
[![GoDoc](https://godoc.org/github.com/invopop/gobl.sa.zatca?status.svg)](https://godoc.org/github.com/invopop/gobl.sa.zatca)
![Latest Tag](https://img.shields.io/github/v/tag/invopop/gobl.sa.zatca)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/invopop/gobl.sa.zatca)

This module implements the Saudi Arabia ZATCA e-invoicing requirements as a
GOBL tax addon (`sa-zatca-v1`), built on top of EN 16931 with KSA-specific
extensions (BR-KSA-* rules). It covers both standard tax invoices (B2B/B2G,
sent for clearance) and simplified tax invoices (B2C, sent for reporting)
through the FATOORA platform.

Unlike the format converters in the GOBL ecosystem, this is a true **addon**:
it registers extensions, normalizers, and validation rules into GOBL's global
registry. It lives in its own module so that only projects handling Saudi
Arabia ZATCA documents take on its weight.

The Saudi Arabia tax regime itself (`regimes/sa`) continues to live in GOBL
core; this module only carries the ZATCA addon.

## Coverage

ZATCA e-invoicing runs through the FATOORA platform and distinguishes two
invoice families:

- **Standard tax invoices** (B2B/B2G) — sent for clearance before issuance.
- **Simplified tax invoices** (B2C) — sent for reporting after issuance.

The addon covers both, along with their associated credit and debit notes,
prepayment invoices, and the export, summary, nominal, third-party and
self-billed variants.

## Layout

- `addon/` — the GOBL addon: extensions, normalizers, scenarios, and
  validation rules that register into GOBL on import. This package is kept
  dependency-light so importing it never pulls in conversion tooling.
- the module root (and future subpackages) is reserved for converters and
  other ZATCA logic built on top of the addon.

## Usage

Add a blank import of the **addon** so it registers itself, then use GOBL as
normal:

```go
import (
	"github.com/invopop/gobl"
	_ "github.com/invopop/gobl.sa.zatca/addon"
)
```

Declare the addon on a document (or let the regime/scenario add it) and
`Calculate` + `Validate` will run the full ZATCA normalization and rules.

> **Note**: the `sa-zatca-v1` key is listed in GOBL core's approved
> external-addon registry, so it is recognised as a valid `$addons` value in
> the JSON Schema. The runtime check stays strict, however: a document
> declaring `sa-zatca-v1` will fail validation with `add-on must be
> registered` unless this module is imported. Any service that processes
> Saudi Arabia ZATCA documents must import it.

## Extensions

| Key | Description |
| --- | --- |
| `sa-zatca-invoice-type` | ZATCA invoice transaction type (KSA-2): a 7-character `TTXNESO` string encoding the main type (`01` standard / `02` simplified) plus binary flags for third-party, nominal, export, summary and self-billed transactions. |
| `vatex-sa` | ZATCA VATEX-SA exemption reason codes, registered by this module's own catalogue (`data/catalogues/vatex_sa.json`). |

VATEX exemption reasons are still carried on documents under GOBL core's CEF
catalogue extension key (`cef-vatex`), but the `VATEX-SA-*` codes themselves
are defined by this module's own catalogue (`data/catalogues/vatex_sa.json`,
registered by `catalogues/vatexsa`): they are ZATCA's KSA extension of the
VATEX code list, not part of the official CEF list. The addon validates them
per VAT category and copies their description into the invoice tax notes
(BR-KSA-83).

## Tags

Set on a `bill.Invoice` to drive the scenarios that populate the KSA-2 code
and the document type:

| Tag | Meaning |
| --- | --- |
| `summary` | Summary invoice |
| `third-party` | Third-party transaction |
| `nominal` | Nominal supply transaction |
| `export` | Export of goods (GOBL core `tax.TagExport`) |
| `simplified` | Simplified tax invoice (GOBL core `tax.TagSimplified`) |
| `self-billed` | Self-billed invoice (GOBL core `tax.TagSelfBilled`) |
| `prepayment` | Prepayment invoice, document type `386` (GOBL core `tax.TagPrepayment`) |

## Document types

The invoice type and tags set the UNTDID document type (BT-3), of which ZATCA
accepts four (BR-KSA-05):

| Type | Tags | BT-3 |
| --- | --- | --- |
| `standard` | | `388` tax invoice |
| `standard` | `prepayment` | `386` prepayment invoice |
| `credit-note` | | `381` credit note |
| `debit-note` | | `383` debit note |

Other tags do not change the code: KSA-2 carries self-billing, and ZATCA has
no partial or factored documents, so a self-billed or factored invoice or
credit note keeps its `388` or `381`. Imported UBL is read back the same way,
and any other BT-3 is kept as stated for BR-KSA-05 to report.

## Validation

Rules register under the `SA-ZATCA` namespace, guarded so they apply only when
`sa-zatca-v1` is active. Fault codes follow GOBL's structured format and the
messages reference the underlying `BR-KSA-*` business rules.

## Development

The addon builds on core GOBL features (the approved external-addon registry
and `pkg/examples` helpers) that are not yet in a tagged release. The
`go.mod` therefore pins `github.com/invopop/gobl` to a commit on the
`addon-sa` branch (a pseudo-version); bump it to the release tag once core is
published.

```sh
go test ./...
```

### Examples

`examples/` holds sample documents covering standard, simplified, credit /
debit notes, self-billed, prepayment and foreign-currency invoices, with their
expected JSON envelopes under `examples/out/`. They are verified via GOBL's shared
`pkg/examples` helpers. Regenerate the golden output after intentional
changes with:

```sh
go test . -run TestExamples -update
```

## Sources

- [ZATCA E-Invoicing Developer Portal](https://zatca.gov.sa/en/E-Invoicing/SystemsDevelopers/Pages/E-Invoice-specifications.aspx)

## License

Apache 2.0 — see [LICENSE](./LICENSE).
