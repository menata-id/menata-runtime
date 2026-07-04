# Menata Runtime — Documentation

## Arsitektur Dokumentasi

```
docs/
├── language/          Menata Language — bahasa untuk domain expert
├── examples/          Contoh lengkap per use case
├── decisions/         ADR — keputusan arsitektur teknis
└── runtime-metadata-schema.md   Referensi schema YAML runtime metadata
```

---

## Untuk Domain Expert

Menulis Business Knowledge → mulai dari sini:

| Dokumen | Isi |
|---------|-----|
| [language/grammar.md](language/grammar.md) | Grammar dan sintaks Menata Language (file `.menata`) |
| [examples/design-request.menata](examples/design-request.menata) | Contoh: Design Request |
| [examples/leave-request.menata](examples/leave-request.menata) | Contoh: Leave Request (HR) |

---

## Untuk Pengembang

Menerjemahkan Business Knowledge ke Runtime Metadata → mulai dari sini:

| Dokumen | Isi |
|---------|-----|
| [runtime-metadata-schema.md](runtime-metadata-schema.md) | Schema lengkap Runtime Metadata (YAML/DB) |
| [examples/design-request.yaml](examples/design-request.yaml) | Contoh: Design Request — YAML realization |
| [examples/leave-request.yaml](examples/leave-request.yaml) | Contoh: Leave Request — YAML realization |
| [examples/README.md](examples/README.md) | Perbandingan dua case + apa yang dibuktikan |

---

## Keputusan Arsitektur (ADR)

| Dokumen | Topik |
|---------|-------|
| [decisions/001-techstack.md](decisions/001-techstack.md) | Pilihan tech stack prototype |
| [decisions/002-metadata-loading.md](decisions/002-metadata-loading.md) | Strategi load metadata + opsi live reload |

---

## Alur Kerja

```
Domain Expert                    Pengembang
─────────────────────            ──────────────────────────────────
1. Tulis .menata           →     2. Terjemahkan ke .yaml
   (grammar.md sebagai referensi)   (runtime-metadata-schema.md sebagai referensi)
                                 3. Buat seeds/*.sql dari .yaml
                                 4. psql -f seeds/*.sql
                                 5. Restart server
```
