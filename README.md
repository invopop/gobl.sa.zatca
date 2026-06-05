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
debit notes, self-billed and foreign-currency invoices, with their expected
JSON envelopes under `examples/out/`. They are verified via GOBL's shared
`pkg/examples` helpers. Regenerate the golden output after intentional
changes with:

```sh
go test . -run TestExamples -update
```

## License

Apache 2.0 — see [LICENSE](./LICENSE).
