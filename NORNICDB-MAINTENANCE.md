# Maintained go-json decoder option

Based on upstream goccy/go-json v0.11.2. Upstream license, source and test
populations are retained. Module/import paths use github.com/orneryd/go-json so
embedding applications consume the pinned dependency without local replace
rules. The encoder remains upstream v0.11.2.

The only behavioral delta is DecodePreserveInvalidUTF8, off by default. The flag
belongs to each decode request or stream; cached decoders carry no mutable mode.
It preserves raw string bytes after normal JSON grammar and escape handling.
Lone escaped surrogates retain upstream behavior; NornicDB validates original
JSON and projects those lexemes before opting in through internal/jsontext.

Review every string normalization path on upgrade. Keep this option's map-key,
typed-field, long-string, streaming/path, concurrent isolation and malformed
input tests, plus the complete unchanged upstream suite. Path.Unmarshal passes
its decode options to the selected-value binder. No storage format, reflection
binder or alternate lexical implementation is added.

First-party regression evidence belongs to NornicDB PR1515. Performance
qualification remains deferred; no benchmark improvement is claimed. Prefer
retiring this maintenance delta when upstream supports the same explicit mode.
