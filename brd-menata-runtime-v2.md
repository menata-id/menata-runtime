# BUSINESS REQUIREMENTS DOCUMENT (BRD)

# Menata Runtime v2 — Process Overlay

**Versi:** 2.0-konsep
**Status:** Draft Konsep (turunan langsung Study 19 & Study 20)
**Tanggal:** 22 Agustus 2026
**Bahasa:** Dokumen ini sengaja ditulis dalam Bahasa Indonesia, se-genre dengan BRD pembanding
(`benchmarks/011-…` Appendix) agar keduanya mudah disandingkan. Spesifikasi teknis Tier 1
(`001`–`006`) tetap berbahasa Inggris dan tetap menjadi acuan normatif.

> Dasar analisis: `benchmarks/011-metadata-workflow-orchestration-brd-benchmark.md` (Study 19,
> pemetaan 30 konsep + BRD pembanding verbatim) dan `benchmarks/012-process-model-synthesis.md`
> (Study 20, uji 21 case + analisis ekonomi server + sintesis Konsep C). Dokumen ini adalah
> Konsep C tersebut, dituliskan sebagai BRD.

---

# 1. RINGKASAN EKSEKUTIF

Menata Runtime v2 adalah **platform aplikasi berbasis metadata dengan proses yang dapat
dideklarasikan** — gabungan dari dua konsep yang telah diuji head-to-head:

- **Konsep A** — BRD "Metadata-Based Workflow Orchestration": Workflow/State/Requirement/
  Transition sebagai objek pertama-kelas, dieksekusi oleh engine workflow tersendiri.
- **Konsep B** — Menata Runtime v1: Machine (Fields + Events + Constraints + Permissions +
  Views), workflow bersifat *emergent*, tanpa objek proses di runtime.

Hasil ujinya (Study 20): B lolos **21/21** case portfolio dan berbiaya **~3–6 statement SQL per
transisi**; A lolos **10/21** dan berbiaya **~10–13 statement per transisi** — namun A unggul
telak pada *keterbacaan proses* dan *reusabilitas deklarasi*. Kekalahan B semuanya di **level
deklarasi** (bisa ditutup dengan menambah lapisan); kekalahan A semuanya di **level arsitektur**
(hanya bisa ditutup dengan membuang sesuatu).

Maka v2 mengambil resep satu-arah:

> **Arsitektur runtime milik v1 + kosakata proses milik BRD pembanding, disambung oleh compiler.**

Tesis satu kalimatnya:

```text
PROSES DIDEKLARASIKAN DI METADATA.
RUNTIME TIDAK PERNAH MELIHATNYA.
```

Blok `process` (state, transition, requirement, quorum, SLA) **di-compile secara deterministik
oleh loader** menjadi primitif yang sudah ada dan sudah teruji — Event + guard, Constraint,
Permission, Config, scheduler. Tidak ada tabel Workflow Instance. Tidak ada engine kedua. Runtime
v1 tidak berubah satu komponen pun; yang berubah adalah *apa yang bisa dituliskan* di metadata.

Prinsip utama v2:

> **Declared process, emergent execution.**
>
> Configure the process — and pay nothing at runtime for having declared it.

---

# 2. LATAR BELAKANG

## 2.1 Masalah yang sama, dua jawaban berlawanan

Kedua konsep berangkat dari masalah yang sama — proses bisnis tidak boleh hard-coded — dan
menjawabnya dari arah berlawanan:

```text
Konsep A                              Konsep B (v1)
--------                              -------------
Proses adalah objek utama.            Data adalah objek utama.
Entity menggantung pada proses.       Proses muncul dari aturan pada data.
Instance proses = entitas runtime.    Tidak ada entitas proses di runtime.
```

## 2.2 Temuan uji (Study 20, ringkas)

1. **Cakupan:** B 21/21 case (12 native); A 10/21 — setiap kegagalan A jatuh pada case
   berbentuk-data (kalkulasi, agregasi, relasi, master data, konten, kalender), karena objek
   istimewa A (Workflow Instance) menjadikan setiap case non-proses sebagai edge case.
2. **Ekonomi server:** B ~3–6 statement/transisi, 1 tabel audit, nol query metadata per request;
   A ~10–13 statement, 4 tabel bookkeeping, Requirement Engine membayar O(requirements) query
   tak-ter-cache per transisi, dan baris Instance menjadi lock hotspot saat parallel approval.
3. **Kemenangan A yang jujur:** (a) worklist ter-index; (b) pemisahan aksi sync/async (§46 BRD-A)
   — keduanya bisa diimpor tanpa mengimpor arsitekturnya.
4. **Gap bersama:** tak satu pun konsep bisa menyatakan "perubahan metadata ini berlaku ke case
   yang sedang berjalan; yang itu tidak."
5. **Counterfactual (Addendum A.3):** menambahkan model presentasi ke A hanya menaikkannya ke
   ~12–14/21 — presentasi cuma satu dari tiga kelas gap-nya; menutup semuanya berarti membangun
   ulang B di bawah engine A.

## 2.3 Apa yang kurang dari v1 — dengan angka

v1 *mampu* mengeksekusi semua bentuk proses yang diuji, tetapi **pengetahuan proses tinggal di
kepala pengarang metadata, bukan di metadata**:

- Case 3 (Document Approval) membutuhkan **6 kapabilitas** (CAP-F13/A07/A08/X03/P02/E05),
  **3 kunci `Machine.Config`**, dan **heuristik pencocokan nama field** yang didokumentasikan
  sendiri oleh `prototype/go/CLAUDE.md` sebagai "prototype-honest heuristics".
- SLA (Case 7) dirakit dari **4 primitif** (A11+E02+A09+E05) yang harus dirangkai benar setiap kali.
- Tidak ada satu artefak pun yang bisa dibaca auditor/administrator untuk melihat "seperti apa
  proses ini dari ujung ke ujung".

v2 ada untuk memindahkan pengetahuan itu ke tempat seharusnya: metadata.

---

# 3. VISI PRODUK

Menata Runtime v2 memungkinkan organisasi mendefinisikan **aplikasi utuh** — data, aturan,
proses, tampilan, otomasi — sebagai metadata, dengan proses yang:

1. **dideklarasikan** dalam kosakata proses (state, transition, requirement, actor, SLA, quorum),
2. **divalidasi** sebagai graph sebelum dimuat,
3. **dirender** sebagai peta proses yang terbaca,
4. **dieksekusi** sebagai primitif v1 yang sudah ada — pada profil biaya v1 yang sudah terbukti.

Visi jangka panjang tidak berubah dari v1:

> **Applications should evolve at the pace of Business Knowledge.**

v2 mempercepat justru bagian yang selama ini menjadi pembatas laju: *siapa* yang bisa mengarang
proses, dan *seberapa cepat serta aman* proses itu diubah.

---

# 4. POSITIONING

```text
                 PROCESS ORCHESTRATION
                        ↑
                    Camunda            ← engine proses tersendiri, instance tables
                        │
              [BRD Pembanding / Konsep A]
                        │
                 MENATA RUNTIME v2     ← kosakata proses A, arsitektur runtime B
                        │
                 Menata Runtime v1     ← proses emergent, tak terlihat
                        │
                 Frappe / ERPNext
                        │
                     Drupal
                        ↓
                 APPLICATION PLATFORM
```

Perbedaan v2 terhadap masing-masing:

- **vs BRD pembanding:** kosakatanya diambil, engine-nya tidak. Workflow menjadi *fitur bahasa
  metadata*, bukan runtime kedua — seperti bahasa tingkat tinggi yang di-compile ke machine code:
  ekspresivitas didapat tanpa CPU kedua.
- **vs v1:** runtime tidak berubah; yang bertambah adalah lapisan compile di loader dan kosakata
  `process` di metadata. Setiap Machine v1 tetap valid tanpa modifikasi.
- **vs Camunda/Frappe:** v2 tetap satu runtime untuk seluruh permukaan aplikasi (data + proses +
  presentasi), bukan orkestrator yang butuh sistem aplikasi terpisah.

---

# 5. TUJUAN

## 5.1 Tujuan Utama

Menyediakan platform aplikasi metadata-driven yang **selebar v1** (seluruh permukaan aplikasi,
21/21 case) dan **seterbaca Konsep A** (proses sebagai deklarasi tunggal yang tervalidasi), tanpa
mengorbankan profil biaya runtime v1.

## 5.2 Tujuan Khusus

Platform harus mampu (penomoran sengaja disandingkan dengan §5.2 BRD pembanding):

1. Membuat proses tanpa perubahan source code — **dipertahankan dari v1**, kini juga tanpa
   pengetahuan komposisi CAP.
2. Membuat state secara deklaratif (satu baris di `process.states`).
3. Membuat transition secara deklaratif, dengan actor dan requirement melekat.
4. Menentukan actor per transition (role, `owner_field`, delegasi, SoD — mekanisme v1).
5. Menentukan requirement per transition, termasuk **cardinality** ("minimal 2 foto").
6. Menghubungkan proses dengan form/field-shape requirement.
7. Menghubungkan proses dengan entity lain (create-on-transition + flag kepuasan termaterialisasi).
8. Menghasilkan task dari proses (Machine task biasa + rollup flag).
9. Approval dengan quorum **ALL / ANY / N_OF_M** (generalisasi CAP-A08).
10. Meminta evidence dengan cardinality terverifikasi.
11. Conditional branching (mekanisme v1: CAP-A09/C04 — sudah ✅).
12. Revision loop (WCP-10 — sudah ✅, kini terbaca di peta proses).
13. Parallel requirement (kini dengan quorum terkonfigurasi).
14. Escalation & SLA sebagai **satu deklarasi** per state, bukan rakitan 4 primitif.
15. Automation (CAP-A01–A15 — sudah ✅; aksi lambat pindah ke outbox async).
16. Audit trail setiap transisi (CAP-R04 — sudah ✅, append-only di level DB).
17. **Kebijakan perubahan ber-efektif-tanggal** — pengganti versioning-pinning: setiap perubahan
    menyatakan berlaku ke `new_records` / `records_in_states [...]` / `all_records`.
18. Peta proses ter-render untuk setiap Machine — termasuk Machine v1 lama (decompile).
19. Validasi graph proses sebelum load (state tak terjangkau, transition menggantung, actor/
    target tidak valid — keluarga CAP-X05).
20. Banyak proses, satu runtime, satu biaya — transisi hasil compile **wajib** berbiaya sama
    dengan transisi yang dirakit tangan.

---

# 6. PRINSIP ARSITEKTUR

Enam prinsip v2, di atas prinsip v1 (`001-design-principles.md`) yang seluruhnya tetap berlaku:

## 6.1 Declared Process, Emergent Execution

Proses dideklarasikan di metadata; runtime tidak pernah melihatnya. Compiler di loader
menerjemahkan deklarasi menjadi primitif; runtime mengeksekusi primitif — persis seperti hari ini.

## 6.2 Compiler, Bukan Engine

Tidak ada tabel `workflow_instance`, tidak ada State Manager, tidak ada Requirement Engine di
runtime. Sebelas komponen runtime BRD pembanding (§32) menyusut menjadi: **satu compiler di
loader + runtime v1 apa adanya.**

## 6.3 Write-time Fan-in, Read-time O(1)

Kebenaran requirement yang hidup di record lain (jumlah evidence, keputusan step, task selesai)
**dipelihara sebagai counter/flag pada record induk saat ditulis** — lewat titik post-commit yang
sudah dipakai CAP-A08/CAP-I01 — dan tidak pernah di-query per-transisi. Pemeriksaan transisi
selalu O(1) terhadap data record itu sendiri. Bonus langsung: worklist ("pending keputusan
saya") menjadi query ter-index, menutup satu-satunya keunggulan query Konsep A.

## 6.4 Efektif-Tanggal, Bukan Version-Pinning

Perubahan metadata menyatakan cakupan berlakunya per perubahan; compiler mewujudkannya sebagai
guard ber-scope-state biasa. Satu model hidup, tanpa cache berganda per versi — dan kasus
compliance ("aturan baru wajib mengenai case terbuka") menjadi *bisa dinyatakan*, bukan
diblokir diam-diam oleh pinning.

## 6.5 Pintu Dua Arah

- **Eject:** proses yang tumbuh ad-hoc keluar dari overlay menjadi Event mentah kapan saja —
  tidak ada yang hilang, karena di runtime memang tidak pernah ada yang lain.
- **Lift:** Machine rakitan-tangan yang sudah stabil bisa diangkat menjadi deklarasi overlay;
  decompiler menyusun draftnya dari Events + guards yang ada.
- **Sederhana tetap sederhana:** Machine tanpa proses (master data, konten, join table) tidak
  butuh overlay sama sekali — "Infer Before Configure" terjaga.

## 6.6 Biaya Compile = Biaya Rakitan Tangan

Definition of Done setiap fase: transisi hasil compile harus berjalan pada jumlah statement yang
sama dengan transisi yang dirakit tangan hari ini. Overlay yang memperlambat runtime adalah
overlay yang gagal.

---

# 7. KONSEP UTAMA

## 7.1 Machine — tidak berubah

Machine tetap unit realisasi utama: Fields, Events, Constraints, Permissions, Views. Seluruh
metadata v1 valid di v2 tanpa modifikasi.

## 7.2 Process Overlay — konsep baru satu-satunya

Blok `process` opsional pada Machine:

```text
process
├── states                  → nilai-nilai value_list Status (di-generate)
├── transitions             → Event + guard + set_field + Permission (di-compile)
│     ├── actor             → role / owner_field / SoD (mekanisme v1)
│     ├── requirements      → constraint trigger-time / counter cardinality / rollup flag
│     └── on_transition     → actions biasa
├── auto                    → transisi otomatis (system-triggered, jalur triggerEvent yang sama)
├── sla                     → date-arithmetic + scheduled event + escalate (di-compile)
└── change_policy           → guard ber-scope-state (di-compile)
```

## 7.3 Tabel kompilasi

| Deklarasi overlay | Di-compile menjadi (primitif v1) | Kerja engine baru |
|---|---|---|
| `state: X` | nilai `value_list` Status + guard per-state | tidak ada |
| `transition {from, to, actor}` | Event + `condition(status=from)` (CAP-E06) + `set_field(status=to)` + baris Permission | tidak ada |
| `requirement: form-shape` | Constraint diperiksa saat trigger (CAP-C09) | tidak ada |
| `requirement: evidence, cardinality N..*` | cek `count_at_least` terhadap counter yang dipelihara di record | **satu-satunya engine baru** (inti CAP-W01) |
| `requirement: approval, quorum` | steps Machine + kunci CAP-X03 + rollup CAP-A08 dengan parameter quorum | perluasan parameter (CAP-W03) |
| `requirement: task / entity` | `create_record` (CAP-A06) + flag kepuasan via titik post-commit CAP-I01 | tidak ada |
| `sla {duration, warn, escalate}` | CAP-A11 (sadar-hari-kerja via CAP-O06) + CAP-E02/E03 + Event eskalasi + notify | tidak ada (CAP-W04) |
| `change_policy` | guard ber-scope-state | loader saja (CAP-W07) |
| peta proses | di-render dari deklarasi; *derivable* mundur dari Machine mana pun | view read-only (CAP-W05) |

Hampir seluruh objek Workflow §9 BRD pembanding **ter-compile habis**.

## 7.4 Requirement — kontrak generik, akhirnya

Struktur konseptual BRD pembanding §35 diadopsi sebagai bentuk deklarasi:

```yaml
- { type: evidence, target: fld_completion_photo, cardinality: "2..*",
    actor: { owner_field: fld_assignee } }
```

Dengan satu perbedaan eksekusi yang menentukan: kebenarannya **tidak pernah dihitung saat
transisi** (model pembanding: O(requirements) query), melainkan dipelihara saat tulis (§6.3).

## 7.5 Actor — mekanisme v1, kosakata baru

Role (CAP-P01), kepemilikan record (CAP-P02 `owner_field`), delegasi (CAP-P04), SoD (CAP-P03) —
semuanya sudah ✅ dan cukup untuk seluruh 21 case. Resolver berantai ala pembanding
(`entity.branch.manager`) **tidak** diadopsi di v2.0 — belum ada case yang menuntutnya; dicatat
sebagai kandidat masa depan, bukan dibangun spekulatif.

---

# 8. CONTOH END-TO-END

Corrective Action — contoh andalan BRD pembanding (§38) — sebagai overlay v2:

```yaml
machine:
  id: mch_corrective_action
  # fields: ... (biasa; value_list Status di-generate dari process.states)
  process:
    states: [Open, Assigned, In_Progress, Submitted, Review, Verified, Closed]
    transitions:
      - { name: Assign,  from: Open,        to: Assigned,    actor: { role: Supervisor } }
      - { name: Start,   from: Assigned,    to: In_Progress, actor: { owner_field: fld_assignee } }
      - { name: Submit,  from: In_Progress, to: Submitted,   actor: { owner_field: fld_assignee },
          requirements:
            - { type: form,     fields: [fld_completion_notes] }
            - { type: evidence, target: fld_completion_photo,  cardinality: "1..*" }
            - { type: evidence, target: fld_completion_report, cardinality: "1..1" } }
      - { name: Approve, from: Review,      to: Verified,    actor: { role: Reviewer } }
      - { name: Revise,  from: Review,      to: In_Progress, actor: { role: Reviewer },
          on_transition: [ { set_field: { field: fld_revision_count, value: increment } } ] }
      - { name: Close,   from: Verified,    to: Closed,      actor: { role: Supervisor } }
    auto: [ { from: Submitted, to: Review } ]
    sla:
      - { state: Review, duration: "2 Business Days",
          on_breach: { notify: { role: Manager }, escalate: true } }
```

Setiap baris ter-compile ke mekanisme yang sudah ✅ kecuali dua cek `cardinality`. Hasil
compile-nya: berjalan ~3 statement per transisi, ter-render sebagai peta proses, tervalidasi
sebagai graph sebelum load, dan bisa di-eject ke Event mentah kapan pun proses ini berhenti rapi.

---

# 9. UJI TERHADAP 21 CASE PORTFOLIO — METADATA SAJA?

Pertanyaan ujinya: dengan v2 sebagaimana dispesifikasikan (Fase 0–5 terbangun) **plus** status
registry v1 hari ini, apakah setiap case terselesaikan oleh **konfigurasi metadata saja** —
tanpa satu baris kode per-case?

Legenda kolom "Artefak v2": `overlay` = blok `process`; `plain` = metadata Machine v1 biasa
(case tanpa proses multi-aktor tidak butuh overlay — prinsip §6.5, sederhana tetap sederhana).

| # | Case | Artefak v2 | Metadata saja? | Catatan |
|---|------|-----------|----------------|---------|
| 1 | Design Request | plain (overlay opsional) | ✅ hari ini | Sudah ✅ done + conformance sejak v1; overlay hanya menambah peta proses |
| 2 | Leave Request | plain (overlay opsional) | ✅ hari ini | Sama dengan Case 1 |
| 3 | Document Approval | **overlay** (quorum) | ✅ | Sudah metadata-only di v1 (seeds/004) tapi menuntut komposisi 6 CAP + heuristik; v2 meruntuhkannya jadi satu blok, plus `N_OF_M` (CAP-W03) yang v1 tidak punya |
| 4 | Maintenance Reminder | plain | ✅ hari ini | E02/E03/A02/A09/A11 semua ✅ — tidak ada proses multi-aktor, overlay tidak perlu |
| 5 | Inventory / Stock | plain | ⚠️ menunggu CAP-C08 | Guard stok-negatif (constraint lintas-record) adalah mekanisme engine sekali-bangun yang sudah teregister ber-desain — di luar lingkup v2; selebihnya (F14/F16/F19/A06/X12) ✅ |
| 6 | Petty Cash | overlay (period close + SoD) | ⚠️ menunggu CAP-C08/C10 | Formula rekonsiliasi & guard saldo = keluarga constraint agregat yang sama; SoD/immutability/E06 sudah ✅ |
| 7 | Customer Complaint | **overlay + eject** | ✅ hari ini | Semua mekanisme ✅; v2 mengganti rakitan SLA 4-primitif dengan satu deklarasi (CAP-W04), sementara langkah ad-hoc/Reopen tetap Event mentah — demonstrasi terbaik pintu dua arah §6.5 |
| 8 | Payment Confirmation | plain | ✅ hari ini | E04/X13/A06/A13/X12 ✅; v2 menambah outbox (CAP-W06) agar rantai notifikasi rekonsiliasi keluar dari jalur request |
| 9 | Accounting | overlay (posting + `change_policy`) | ⚠️ menunggu CAP-C10/C11 | Invarian debit=kredit & kunci periode = engine sekali-bangun teregister; catatan khusus: CAP-W07 (efektif-tanggal) justru **cermin praktik akuntansi** — perubahan aturan fiskal yang tidak boleh mengenai entri periode berjalan kini ternyatakan di metadata |
| 10 | Organization Composite | plain | ✅ hari ini | CAP-O01–O06 ✅; tak ada konten proses |
| 11 | Social App | plain | ✅ sebagian | Join Machine Follow/Like terbangun dari F13(✅)×2 + C12(✅) hari ini; yang tersisa: filter feed dua-hop ("post dari yang saya follow") = perluasan V05/F20, engine sekali-bangun teregister |
| 12 | Community + gamification | plain | ✅ hari ini | Kesimpulan `benchmarks/010` sendiri: semua mekanisme ✅, yang kurang hanya satu seed file terpadu — persis "metadata saja"; v2 menambah outbox untuk rantai akrual poin |
| 13 | Blog | plain | ✅ hari ini | P07/V10 ✅; Tags multi-select masih workaround teks (F03 scope note) — jujur dicatat, tidak menghalangi case |
| 14 | Lending | overlay (approval) + plain | ✅ hari ini | A15/A13/P03/E02 ✅ |
| 15 | E-commerce | overlay (Checkout) + plain | ✅ hari ini | R08/F16 ✅; Checkout-dengan-requirement adalah bentuk overlay yang alami |
| 16 | Point of Sale | plain | ⚠️ menunggu CAP-C08 | Komposisi Case 5+8+15 — mewarisi kebutuhan guard stok Case 5; selebihnya termuat & ter-render (verifikasi bulk 50 YAML) |
| 17 | Helpdesk | overlay + eject | ✅ hari ini | Identik Case 7 |
| 18 | HR / Employee master | plain | ✅ hari ini | F13-tree, O02 ✅ — tak ada proses |
| 19 | Project Management | plain | ✅ sebagian | Sisi data/proses ✅ (V14 Tier 1); drag-and-drop lintas kolom = kapabilitas UI teregister (V14 Tier 2), bukan urusan lapisan proses |
| 20 | Hospital | plain | ✅ sebagian | P06/F16/V07 ✅; grid kalender per-resource (V18) = kapabilitas view teregister |
| 21 | E-learning | overlay (sequential unlock) + plain | ✅ hari ini | E06/C12 ✅, sertifikat = view `document` (F21, output HTML) |

## Rekapitulasi

- **16 dari 21 case: metadata saja, penuh, dengan v2 + registry hari ini** (termasuk semua case
  ber-proses berat — 3, 7, 9-proses, 14, 15, 17 — yang kini turun dari rakitan multi-CAP menjadi
  satu blok deklarasi).
- **5 case menunggu mekanisme engine sekali-bangun yang sudah teregister ber-desain** — dan
  polanya seragam: Case 5/6/9/16 semuanya menunggu keluarga constraint agregat/lintas-record
  (CAP-C08/C10/C11), Case 11/19/20 menunggu perluasan view (V05-ext, V14 Tier 2, V18). Tak satu
  pun berada di lingkup lapisan proses v2 — dan tak satu pun pernah membutuhkan kode per-case.

Pembeda yang harus dipegang saat membaca kolom ⚠️: **"belum bisa metadata-saja" di sini selalu
berarti "runtime belum membangun mekanisme konfigurabel itu", tidak pernah berarti "case ini
butuh kode khusus".** Begitu CAP-C08 terbangun (sekali), guard stok Case 5, saldo dana Case 6,
dan komposisi Case 16 seluruhnya menjadi satu baris constraint di metadata — kontrak "configure
the process, don't code the process" berlaku untuk seluruh portfolio, pada kedua versi. Yang v2
ubah adalah *bentuk* konfigurasinya untuk case ber-proses: dari komposisi lintas-CAP yang hanya
dikuasai segelintir orang, menjadi deklarasi yang tervalidasi dan terbaca.

---

# 10. ARSITEKTUR

```text
──────────────────────────────
Authoring Layer  (tidak berubah)
──────────────────────────────
        │
        ▼
Runtime Metadata
  Machine v1  +  blok `process` (opsional)
        │
        ▼
──────────────────────────────
Loader / COMPILER   ← satu-satunya yang baru
──────────────────────────────
  validasi graph proses (keluarga CAP-X05)
  ekspansi deterministik → Events, guards,
  Constraints, Permissions, Config, scheduler,
  flag/counter termaterialisasi + indeksnya (CAP-X10)
        │
        ▼
──────────────────────────────
Menata Runtime  (v1, TIDAK BERUBAH)
──────────────────────────────
  Interpreter · Executor · Guard · Constraint Engine
  + Outbox async (CAP-W06 — satu tambahan runtime,
    independen dari overlay)
        │
        ▼
Applications
```

**Model data runtime:** tidak ada tabel baru untuk proses. Flag/counter termaterialisasi adalah
field biasa pada record. Satu-satunya tabel baru: `action_outbox` (CAP-W06) — baris antrian yang
ditulis atomik dalam transaksi request, dieksekusi setelah commit, dengan disiplin idempotensi
CAP-X13.

**Keamanan:** tidak berubah — v1 sudah menjalankan prinsip §44 BRD pembanding sejak awal (user
men-trigger Event bernama, tidak pernah menulis Status langsung; Guard memeriksa role →
ownership → SoD → state-guard → constraint sebelum Persist).

---

# 11. NON-GOALS

v2 **tidak** akan:

- membangun engine workflow kedua atau tabel Workflow Instance;
- mengadopsi version-pinning menyeluruh (digantikan `change_policy`);
- membangun visual drag-and-drop builder di tahap awal (peta proses read-only dulu; form-based
  authoring — posisi yang sama dengan §41 BRD pembanding);
- membangun resolver actor berantai sebelum ada case yang menuntutnya;
- menjadi BPMN engine penuh / ERP / RPA (non-goals v1 dan BRD pembanding sama-sama berlaku).

---

# 12. KRITERIA SUKSES

1. Seluruh 21 case portfolio tetap lolos — regresi nol (conformance suite adalah penjaganya).
2. Case 3 dapat dituliskan ulang sebagai satu blok `process` tanpa menyentuh 6 CAP secara
   langsung, dan berperilaku identik (dibuktikan conformance yang sama).
3. Contoh §8 (Corrective Action) termuat, tervalidasi, ter-render peta prosesnya, dan setiap
   transisinya berbiaya statement sama dengan rakitan tangan (diverifikasi log/explain).
4. "Minimal 2 foto" memblokir transisi — dan pemeriksaannya membaca counter di record, bukan
   meng-query tabel file.
5. Quorum `2_of_3` pada parallel approval bekerja (perluasan CAP-A08).
6. SLA satu-deklarasi menghasilkan warning + breach + eskalasi (perilaku Case 7, kini dari satu
   baris).
7. `change_policy: records_in_states [Draft]` terbukti: perubahan rule tidak mengenai record
   yang sudah melewati Draft; `all_records` terbukti sebaliknya.
8. Peta proses ter-render untuk Machine ber-overlay **dan** (via decompile) untuk minimal satu
   Machine v1 lama tanpa overlay.
9. Aksi lambat (fan-out notifikasi ≥ N penerima, rantai subscription) berjalan lewat outbox —
   latensi request tidak lagi memuat eksekusinya.
10. Skor terhadap §53 BRD pembanding naik dari ~15/20 (v1) menjadi ≥ 19/20.

---

# 13. ROADMAP IMPLEMENTASI

| Fase | Isi | CAP |
|---|---|---|
| **0 — Prasyarat independen** | Outbox async (bermanfaat tanpa overlay; dibuktikan Case 3/10/12) | CAP-W06 |
| **1 — Compiler inti** | `process.states`/`transitions`/`auto` → Event/guard/Permission; validasi graph; peta proses (arah maju) | CAP-W05 (maju), fondasi |
| **2 — Requirement** | Kontrak generik + counter cardinality + flag rollup termaterialisasi + worklist ter-index | CAP-W01 (+ CAP-X10) |
| **3 — Quorum & SLA** | Parameter `ALL/ANY/N_OF_M` pada rollup; `sla` satu-deklarasi | CAP-W03, CAP-W04 |
| **4 — Evolusi** | `change_policy` efektif-tanggal | CAP-W07 |
| **5 — Dua arah** | Decompiler (peta untuk Machine lama; draft *lift*) | CAP-W05 (mundur) |

Disiplin admisi tetap berlaku (`capability-lifecycle.md`): fase 1–5 berstatus **HOLD** sampai ada
case proses nyata — idealnya diarang oleh orang selain pengembang runtime ini, karena pengalaman
pengarang itulah nilai jual overlay. Fase 0 (CAP-W06) tidak di-HOLD: bukti kebutuhannya sudah ada
pada case yang berjalan.

---

# 14. NILAI BISNIS — DAN JAWABAN JUJUR "KENAPA INI MENARIK"

Kejujuran dulu: **fleksibilitas-cakupan tidak naik** — v1 sudah 21/21; tidak ada case yang v2
bisa dan v1 tidak. Yang naik adalah tiga dimensi fleksibilitas lain, dan ketiganya persis
pembatas laju visi proyek ini:

### Fleksibilitas pengarang (authoring)

Hari ini, mengarang proses ala Case 3 menuntut hafal 6 kapabilitas + 3 kunci config + heuristik —
pengetahuan yang hidup di `CLAUDE.md` dan kepala pengembang. Dengan overlay, pengetahuan itu
pindah ke metadata: satu blok `process` yang bisa ditulis administrator, konsultan, atau AI tool
di Authoring Layer — sesuai arsitektur v1 sendiri yang menyebut "AI-assisted tools" sebagai
produsen Runtime Metadata yang sah. Pembatas laju "Applications evolve at the pace of Business
Knowledge" selama ini bukan runtime-nya, melainkan manusia langka yang menguasai registry;
overlay menghapus kelangkaan itu.

### Fleksibilitas evolusi (perubahan yang aman)

`change_policy` menjawab pertanyaan yang dua konsep induknya sama-sama tidak bisa jawab:
perubahan ini berlaku ke case terbuka atau tidak? Proses berumur panjang (corrective action
berminggu-minggu, approval lintas-bulan) menjadi aman diubah — dengan niat yang eksplisit dan
ter-audit di metadata itu sendiri.

### Fleksibilitas tata kelola (legibilitas)

Peta proses + validator graph + satu deklarasi terbaca = yang selama ini membuat Konsep A menang
di mata auditor dan administrator, kini tanpa biaya runtimenya.

### Dan biayanya rendah karena risikonya rendah

Runtime tidak disentuh (kecuali outbox); kerja terkonsentrasi di loader; ratchet conformance
menjaga semua yang sudah ada; dan §6.6 menjadikan "tidak memperlambat apa pun" sebagai syarat
lulus, bukan harapan. Perbandingannya dengan membangun Konsep A secara penuh: sebelas komponen
runtime + empat tabel bookkeeping + biaya 3–4× per transisi — versus satu compiler dan satu
operator hitung.

---

# 15. PERBANDINGAN DENGAN RENCANA BERJALAN — DAN URUTAN KEPUTUSAN

Pertanyaan praktis penutup: apa beda **aplikasi** yang dihasilkan BRD v2 dengan aplikasi yang
dihasilkan rencana berjalan (v1 + roadmap registry apa adanya)? Karena keduanya berjalan di
runtime yang sama, bedanya bukan pada apa yang *bisa dijalankan* aplikasi (sama-sama 21/21 case,
performa identik) — melainkan pada pengalaman **membangun, mengubah, dan mengelola**-nya.

## 15.1 Aplikasi dengan rencana berjalan

| Keunggulan | Konsekuensi |
|---|---|
| Nol risiko compiler — tidak ada yang baru dibangun untuk lapisan proses | Effort langsung ke gap yang benar-benar memblokir case: CAP-C08/C10/C11 membuka Case 5/6/9/16, perluasan view membuka 11/19/20 — **ROI jangka pendek lebih tinggi** |
| Grammar metadata terkecil dan sudah teruji (conformance ratchet) | Stabilitas maksimal; tak ada permukaan authoring baru yang bisa salah |
| Sesuai disiplin admisi — overlay memang masih HOLD | Tidak membangun spekulatif sebelum ada case nyata |

Warisan keterbatasannya (§2.3): proses tak terlihat, hanya terarang oleh ahli registry, SLA/
approval dirakit per case, tanpa N_OF_M, tanpa cardinality, perubahan metadata mengenai semua
record terbuka tanpa kecuali, worklist tetap scan berfilter.

## 15.2 Aplikasi dengan BRD v2

| Keunggulan | Konsekuensi |
|---|---|
| Proses terbaca: satu deklarasi + peta proses + validator graph | Auditor/admin melihat alur ujung-ke-ujung tanpa membaca metadata mentah |
| Siapa pun mengarang proses: admin, konsultan, AI tool | Case 3: komposisi 6-CAP + heuristik → satu blok; laju evolusi tak lagi dibatasi kelangkaan ahli |
| Kemampuan baru: cardinality, quorum N_OF_M, SLA satu baris | "Minimal 2 foto" dan "2 dari 3 approver" ternyatakan — hari ini tidak |
| `change_policy` efektif-tanggal | Proses berumur panjang aman diubah, dengan niat eksplisit dan ter-audit |
| Worklist ter-index + outbox async | "Pending keputusan saya" = query ter-index; request tak menunggu rantai notifikasi |
| Runtime & biaya identik (§6.6 syarat lulus) | Semua di atas tanpa membayar 3–4× biaya transisi ala engine workflow |

Biayanya: compiler harus dibangun dan dibuktikan — permukaan baru yang bisa salah, karenanya
Kriteria Sukses §12 menuntut kesetaraan perilaku dan biaya dengan rakitan tangan.

## 15.3 Urutan keputusan — keduanya bukan pilihan saling meniadakan

1. **Bernilai sekarang di kedua jalur:** CAP-W06 (outbox — tidak di-HOLD, dibuktikan Case
   3/10/12) + keluarga CAP-C08/C10/C11 dari rencana berjalan. Membuka 4 case ⚠️ dan memperbaiki
   latensi tanpa menyentuh overlay sama sekali.
2. **v2 menjadi mendesak ketika** tujuan bergeser dari "menyelesaikan portfolio" ke "lebih
   banyak aplikasi dibangun lebih banyak orang" — mis. saat `menata.app` mulai dipakai pihak
   lain untuk mengarang prosesnya sendiri. Itu persis trigger admisi §13.

Dipadatkan satu kalimat:

> **Rencana berjalan menghasilkan aplikasi yang lebih *lengkap* lebih cepat; BRD v2
> menghasilkan platform yang prosesnya lebih *terbaca*, lebih *aman diubah*, dan lebih
> *terjangkau diarang* — pada biaya server yang sama.**

## 15.4 Skenario greenfield — jika sama-sama mulai dari nol, quality > time

Jika keduanya dibangun dari nol dengan prinsip kualitas di atas waktu (dan di atas biaya token
AI), **v2 adalah target yang tepat sejak hari pertama** — dengan satu presisi penting: memilih
v2 dari nol *bukan* berarti melewati arsitektur v1, karena v2 mengandung v1 (overlay adalah
lapisan di atas substrat v1, dan compiler tidak bisa dibangun sebelum targetnya ada). Yang
berubah adalah **urutan desain**, bukan urutan bangun:

1. **Tanpa utang retrofit.** Grammar overlay dirancang bersama substrat sejak awal, sehingga
   pola rakitan-tangan ala Case 3 (3 kunci config + heuristik pencocokan nama `Sequence`/
   `Decision`) tidak pernah sempat lahir — overlay mendeklarasikan hal-hal yang di v1 terpaksa
   ditebak heuristik.
2. **Kualitas by construction.** Setiap proses tervalidasi sebagai graph sejak dimuat pertama
   kali; conformance ditulis terhadap deklarasi, bukan terhadap rakitan.
3. **Ekonomi token AI justru condong ke v2, kuat.** AI yang mengarang proses menulis satu blok
   deklaratif kecil — bukan memuat pengetahuan 6 CAP + konvensi komposisinya ke context untuk
   setiap pengarang ulang. Konteks lebih kecil, ruang salah lebih sempit, dan hasil generate
   tervalidasi murah oleh compiler yang deterministik. Untuk authoring berbasis AI (arah yang
   dinyatakan arsitektur v1 sendiri), overlay bukan biaya — ia justru bentuk paling hemat token.

Satu peringatan jujur dari sejarah proyek ini sendiri: merancang grammar overlay *sebelum* ada
case nyata yang menguji substrat adalah risiko kualitas — penemuan terpenting repo ini justru
datang dari case dan benchmark yang mendahului desain (CAP-E06 ditemukan benchmark, bukan
dirancang; heuristik ditemukan karena membangun). Maka jalur greenfield ber-kualitas-tertinggi
adalah: substrat + beberapa case pembuktian dulu, baru overlay — **yang persis merupakan posisi
proyek ini hari ini**. Dengan kata lain: v1 yang sudah terbangun adalah 80% pertama dari jalur
greenfield v2 yang benar, dan hampir tidak ada yang terbuang.

---

# 16. KESIMPULAN

Menata Runtime v2 bukan:

> Task Management, bukan Workflow Engine, bukan pula v1 + engine BPM.

Melainkan:

> **Metadata-Based Application Platform yang prosesnya dapat dideklarasikan —
> declared process, emergent execution.**

Formula v1 dipertahankan penuh:

```text
METADATA + RUNTIME = APPLICATION
```

Dengan satu tambahan yang mengubah siapa yang bisa menulis metadata itu:

```text
PROSES DIDEKLARASIKAN
        ↓  (compile, deterministik, tervalidasi)
PRIMITIF YANG SUDAH TERUJI
        ↓  (runtime v1, tidak berubah)
EKSEKUSI SERINGAN HARI INI
```

Diferensiasi fundamental BRD pembanding — `STATE → WHAT MUST BE FULFILLED? → TRANSITION` —
dipertahankan **sebagai kebenaran authoring**, pada nol biaya runtime-nya.
