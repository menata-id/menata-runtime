# Menata Runtime (Go Prototype) — Documentation

Dokumentasi di folder ini khusus untuk prototype Go. Panduan bahasa dan Runtime Metadata ada di level yang lebih tinggi.

---

## Panduan Penulisan

| Dokumen | Untuk siapa | Isi |
|---------|-------------|-----|
| [`guides/writing-menata.md`](../../../../guides/writing-menata.md) | Domain expert | Cara menulis `.menata` dari nol |
| [`runtime/guides/writing-runtime-metadata.md`](../../guides/writing-runtime-metadata.md) | Developer | Cara menerjemahkan `.menata` ke YAML → SQL |
| [`runtime/guides/runtime-metadata-gotchas.md`](../../guides/runtime-metadata-gotchas.md) | Developer | Gotcha loader + checklist sebelum seed |
| [`runtime/guides/writing-process-overlays.md`](../../guides/writing-process-overlays.md) | Developer | Jalan pintas `process` block |

---

## Referensi Schema

| Dokumen | Isi |
|---------|-----|
| [../../../runtime-metadata-schema.md](../../../runtime-metadata-schema.md) | Schema lengkap Runtime Metadata (YAML/DB) — semua field, tipe, contoh |

---

## Contoh Lengkap

| Dokumen | Isi |
|---------|-----|
| [examples/README.md](examples/README.md) | Perbandingan dua case + apa yang dibuktikan |
| [examples/design-request.menata](examples/design-request.menata) | Case 1: Design Request — Menata Language |
| [examples/design-request.yaml](examples/design-request.yaml) | Case 1: Design Request — Runtime Metadata |
| [examples/leave-request.menata](examples/leave-request.menata) | Case 2: Leave Request — Menata Language |
| [examples/leave-request.yaml](examples/leave-request.yaml) | Case 2: Leave Request — Runtime Metadata |

---

## Keputusan Arsitektur (ADR)

| Dokumen | Topik |
|---------|-------|
| [decisions/001-techstack.md](decisions/001-techstack.md) | Pilihan tech stack prototype |
| [decisions/002-metadata-loading.md](decisions/002-metadata-loading.md) | Strategi load metadata + opsi live reload |
| [decisions/003-tenancy-and-indexing.md](decisions/003-tenancy-and-indexing.md) | Strategi tenancy + indexing |
| [decisions/004-internal-package-architecture.md](decisions/004-internal-package-architecture.md) | Target layout `internal/` untuk extension seams (migrasi capability-triggered) |
| [decisions/005-deployment-status.md](decisions/005-deployment-status.md) | Status deployment `menata.app` — cakupan NFR yang sudah live |
| [decisions/006-handler-file-split.md](decisions/006-handler-file-split.md) | Audit panjang baris/file + split `handler.go` per domain |
| [decisions/007-conformance-suite-split.md](decisions/007-conformance-suite-split.md) | Split `conformance/run.sh` (2128 baris) menjadi `lib.sh` + `tests/NNN_*.sh`, verifikasi behavioral equivalence di dua schema fresh terisolasi |
