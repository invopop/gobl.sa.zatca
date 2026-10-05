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

The module is two things at once. The `addon/` subpackage is a true **addon**:
it registers extensions, normalizers, and validation rules into GOBL's global
registry, and is kept dependency-light. The module root is the **converter**,
turning GOBL documents into ZATCA UBL 2.1 and back, built on top of
[gobl.ubl](https://github.com/invopop/gobl.ubl) for the generic EN 16931
plumbing. It lives in its own module so that only projects handling Saudi
Arabia ZATCA documents take on its weight.

The Saudi Arabia tax regime itself (`regimes/sa`) continues to live in GOBL
core; this module only carries the ZATCA addon.

## Coverage

ZATCA e-invoicing runs through the FATOORA platform and distinguishes two
invoice families:

- **Standard tax invoices** (B2B/B2G) — sent for clearance before issuance.
- **Simplified tax invoices** (B2C) — sent for reporting after issuance.

The addon covers both, along with their associated credit and debit notes and
the export, summary, nominal, third-party and self-billed variants.

## Layout

- `addon/` — the GOBL addon: extensions, normalizers, scenarios, and
  validation rules that register into GOBL on import. This package is kept
  dependency-light so importing it never pulls in conversion tooling.
- the module root — the GOBL ⇄ ZATCA UBL converter, which pulls in gobl.ubl.

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

## Conversion

`Convert` builds the ZATCA document, `Bytes` renders it:

```go
import (
	"github.com/invopop/gobl"
	zatca "github.com/invopop/gobl.sa.zatca"
)

doc, err := zatca.ConvertInvoice(env)
data, err := zatca.Bytes(doc)
```

An incoming ZATCA document goes the other way:

```go
doc, err := zatca.ParseInvoice(data)
env, err := doc.Convert()
```

The converter runs the generic EN 16931 mapping in gobl.ubl and reworks the
result into ZATCA Phase 2: the KSA-2 transaction type, the document UUID and
issue time, the SAR accounting currency and its repeated tax total, the KSA
address fields, the line VAT amounts (KSA-11/KSA-12), and the credit and debit
notes ZATCA carries in a UBL `Invoice` rather than a `CreditNote`.

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

Set on a `bill.Invoice` to drive the scenario that populates the KSA-2 code:

| Tag | Meaning |
| --- | --- |
| `summary` | Summary invoice |
| `third-party` | Third-party transaction |
| `nominal` | Nominal supply transaction |
| `export` | Export of goods (GOBL core `tax.TagExport`) |
| `simplified` | Simplified tax invoice (GOBL core `tax.TagSimplified`) |
| `self-billed` | Self-billed invoice (GOBL core `tax.TagSelfBilled`) |

## Validation

Rules register under the `SA-ZATCA` namespace, guarded so they apply only when
`sa-zatca-v1` is active. Fault codes follow GOBL's structured format and the
messages reference the underlying `BR-KSA-*` business rules.

## Development

```sh
go test ./...
```

### Schematron validation

Beyond the golden-file comparisons, every converted document can be pushed
through ZATCA's own schematron rule set. Validation runs against
[phorm](https://github.com/phax/phorm), the standalone validation service.
Start one locally:

```sh
docker run -d --name phorm -p 8080:8080 phelger/phorm
```

Use `phelger/phorm-arm64` on Apple Silicon. It takes a few seconds to boot; it
is ready once this returns HTTP 200:

```sh
curl -s -o /dev/null -w '%{http_code}\n' \
  -H 'X-Token: phorm-dev-token' \
  'http://localhost:8080/api/get/vesids?include-deprecated=true'
```

Then run the suite with `-validate`:

```sh
go test . -validate
```

Without `-validate` the validating tests are skipped, so the plain `go test
./...` never needs a service. `PHORM_URL` and `PHORM_TOKEN` override the
defaults (`http://localhost:8080` and phorm's stock development token).

#### Known failures

`BR-KSA-33` fails on every document: the invoice counter value (KSA-16) is
stamped by the application when the document is submitted, not carried in
GOBL, so a converted document never has one. The test ignores it by name.

### Examples

`examples/` holds sample documents covering standard, simplified, credit /
debit notes, self-billed, exempt, tax-inclusive and foreign-currency invoices.
Each stage feeds the next, so a document is carried all the way out to ZATCA
and back:

| Stage | Test | In | Out |
| --- | --- | --- | --- |
| Build | `TestExamples` | `examples/*.yaml` | `examples/out/*.json` |
| Convert | `TestConvert` | `examples/out/*.json` | `test/data/convert/out/*.xml` |
| Parse | `TestParse` | `test/data/convert/out/*.xml` | `test/data/parse/out/*.json` |

Nothing is copied between stages: each test reads the previous one's golden
directory, so `examples/` stays GOBL-only and an example added there is covered
end to end. Regenerate the goldens in order after an intentional change:

```sh
go test . -run TestExamples -update
go test . -run TestConvert -update
go test . -run TestParse -update
```

## Sources

- [ZATCA E-Invoicing Developer Portal](https://zatca.gov.sa/en/E-Invoicing/SystemsDevelopers/Pages/E-Invoice-specifications.aspx)

## License

Apache 2.0 — see [LICENSE](./LICENSE).
