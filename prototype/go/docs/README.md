# Menata Runtime (Go Prototype) — Documentation

Dokumentasi di folder ini khusus untuk prototype Go. Panduan bahasa dan Runtime Metadata ada di level yang lebih tinggi.

---

## Panduan Penulisan

| Dokumen | Untuk siapa | Isi |
|---------|-------------|-----|
| [`guides/writing-menata.md`](../../../../guides/writing-menata.md) | Domain expert | Cara menulis `.menata` dari nol |
| [`runtime/guides/writing-runtime-metadata.md`](../../guides/writing-runtime-metadata.md) | Developer | Cara menerjemahkan `.menata` ke YAML → SQL |

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
