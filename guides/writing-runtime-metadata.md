# Panduan Menulis Runtime Metadata

Panduan ini menjelaskan cara menerjemahkan file `.menata` menjadi Runtime Metadata — dari YAML sampai ke SQL yang siap dijalankan.

**Untuk siapa:** Developer yang mengimplementasikan machine baru ke Menata Runtime.  
**Prasyarat:** Sudah ada file `.menata` yang ditulis domain expert (lihat panduan di
[`menata-id/menata`](https://github.com/menata-id/menata/tree/main/guides) — repo terpisah untuk
layer bahasa proses bisnis, tidak ada kaitan dengan mesin/aplikasi).

Referensi schema lengkap: `runtime/runtime-metadata-schema.md`  
Cara memilih tipe field yang tepat (primitif vs `value_list` vs `reference`): `runtime/benchmarks/005-field-modeling-decision-framework.md`

---

## Posisi Runtime Metadata dalam Arsitektur

```
Business Knowledge (.menata)
        │
        ▼
Runtime Metadata (YAML → SQL)
        │
        ▼
Menata Runtime (Go prototype)
        │
        ▼
Application
```

File `.menata` mendeskripsikan **apa yang bisnis tahu**.  
Runtime Metadata mendeskripsikan **bagaimana runtime merealisasikannya**.

Keduanya berbeda. Jangan menulis concern runtime di `.menata`, dan jangan menulis Business Knowledge ulang di Runtime Metadata.

---

## Hierarki

```
Workspace
    └── Application
            └── Machine
                    ├── Fields
                    ├── Events
                    │       └── Event Actions
                    ├── Constraints
                    ├── Permissions
                    └── Views
```

Satu Machine = satu Object dari `.menata`.

---

## Konvensi ID

Setiap elemen Runtime Metadata memiliki ID stabil. ID **tidak boleh berubah** setelah diassign — nama boleh berubah, ID tidak.

| Elemen | Prefix | Contoh |
|--------|--------|--------|
| Workspace | `ws_` | `ws_default` |
| Application | `app_` | `app_hr` |
| Machine | `mch_` | `mch_leave_request` |
| Field | `fld_` | `fld_lr_reason` |
| Event | `evt_` | `evt_lr_submit` |
| Constraint | `cst_` | `cst_lr_reason_required` |
| Permission | `perm_` | `perm_lr_employee` |
| View | `vw_` | `vw_lr_form` |

Untuk machine dengan banyak elemen, tambahkan infix singkat setelah prefix agar tidak bentrok antar machine:

```
fld_lr_*    → fields untuk mch_leave_request (lr = leave request)
fld_dr_*    → fields untuk mch_design_request (dr = design request)
```

ID ditulis dalam `snake_case`, huruf kecil semua.

---

## Pemetaan: .menata → YAML → SQL

### Object → Machine

**.menata**
```
Leave Request
```

**YAML**
```yaml
machine:
  id: mch_leave_request
  name: Leave Request
  application: app_hr
```

**SQL**
```sql
INSERT INTO machines (id, application_id, name) VALUES
    ('mch_leave_request', 'app_hr', 'Leave Request');
```

#### Machine `config` — opsional, jarang dibutuhkan

Kosong (`NULL`) untuk hampir semua Machine. Isi hanya kalau ada mekanisme runtime yang butuh tahu
sesuatu tentang Machine ini sendiri — bukan Field, bukan Constraint, murni "cara Machine ini
berperilaku" (CAP-X03). Setiap kapabilitas baru yang butuh setelan level-Machine menambah kunci
baru di sini, bukan kolom migrasi baru. Kunci yang ada per 2026-07-12:

| Kunci | Dipakai oleh | Arti |
|-------|--------------|------|
| `approval_mode_field`, `steps_machine`, `steps_parent_field` | CAP-A07, CAP-A08 | bentuk workflow sequential/aggregate, lihat §Events dan Actions |
| `webhook_secret` | CAP-E04 | kredensial yang harus dibawa `POST /webhooks/...` di header `X-Webhook-Secret` |
| `master_data` (`"true"`) | CAP-O02 | menandai Machine ini kanonis/dirujuk lintas-Application — Archive diblokir selama masih ada record LAIN, di Machine mana pun, yang merujuknya |
| `immutable_field`, `immutable_values` (dipisah koma) | CAP-R07 | begitu Field ini bernilai salah satu `immutable_values`, record menolak Update maupun Archive |
| `scratch_field`, `scratch_values` (dipisah koma) | CAP-R08 | record yang dibuat dengan Field ini sudah bernilai salah satu `scratch_values` melewati Constraint yang normalnya memblokir, sampai keluar dari state itu |
| `sod_reference_field`, `sod_requester_field` | CAP-P03 | Segregation of Duties di Machine anak/pemutus: `sod_reference_field` = field `reference` ke parent; `sod_requester_field` = field `type: user` di parent yang menyimpan siapa pengaju — identity yang login tidak boleh sama dengan nilai itu |

```sql
UPDATE machines SET config = '{
  "approval_mode_field": "fld_ad_approval_mode",
  "steps_machine": "mch_approval_step",
  "steps_parent_field": "fld_as_document"
}' WHERE id = 'mch_approval_document';

UPDATE machines SET config = '{"master_data": "true"}' WHERE id = 'mch_wsx_employee';
```

Kalau Machine Anda tidak butuh ini, jangan diisi — bukan bagian wajib.

---

### Fields

**.menata**
```
Fields

- Employee : User
- Leave Type : Annual Leave | Sick Leave | Emergency Leave | Unpaid Leave
- Start Date : Date
- Reason : Rich Text
```

**YAML**
```yaml
fields:
  - id: fld_lr_employee
    name: Employee
    type: user
    required: true

  - id: fld_lr_leave_type
    name: Leave Type
    type: value_list
    required: true
    values:
      - Annual Leave
      - Sick Leave
      - Emergency Leave
      - Unpaid Leave

  - id: fld_lr_start_date
    name: Start Date
    type: date
    required: true

  - id: fld_lr_reason
    name: Reason
    type: rich_text
    required: true
```

**SQL**
```sql
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_lr_employee',   'mch_leave_request', 'Employee',   'user',       0, true, '{}'),
    ('fld_lr_leave_type', 'mch_leave_request', 'Leave Type', 'value_list', 1, true,
        '{"values":["Annual Leave","Sick Leave","Emergency Leave","Unpaid Leave"]}'),
    ('fld_lr_start_date', 'mch_leave_request', 'Start Date', 'date',       2, true, '{}'),
    ('fld_lr_reason',     'mch_leave_request', 'Reason',     'rich_text',  3, true, '{}');
```

#### Tipe Field — pemetaan

| Di .menata | `type` di DB | `options` |
|------------|-------------|-----------|
| `Text` | `text` | `{}` |
| `Rich Text` | `rich_text` | `{}` |
| `Number` | `number` | `{}` |
| `Money` | `money` | `{"currency":"IDR"}` — **wajib**, lihat catatan di bawah |
| `Boolean` | `boolean` | `{}` |
| `Date` | `date` | `{}` |
| `Time` | `time` | `{}` |
| `Date Time` | `date_time` | `{}` |
| `Duration` | `duration` | `{}` |
| `User` | `user` | `{}` |
| `File` | `file` | `{}` |
| `A \| B \| C` | `value_list` | `{"values":["A","B","C"]}` |
| `Reference: X` | `reference` | `{"target_machine":"mch_x"}` |
| — | `computed` | `{"source_field":"fld_x","factor":1.1}` atau `{"source_field":"fld_x","factor_field":"fld_y"}` — CAP-F14 ✅, tidak pernah tersimpan, dihitung ulang tiap render |

> ⚠️ **Key yang benar untuk `reference` adalah `target_machine`, bukan `machine`.** Konsisten dengan
> contoh nyata di `prototype/go/docs/examples/approval-step.yaml` (`Document` field, `mch_approval_document`).

**Status per 2026-07-12 (batch terakhir)**: `number` (CAP-F07), `money` (CAP-F08), `boolean`
(CAP-F09), `time`/`date_time` (CAP-F10) semua ✅ — render sebagai input HTML5 sungguhan
(`type="number"`/`"checkbox"`/`"time"`/`"datetime-local"`), bukan lagi text fallback.
`duration` (CAP-F10) disimpan sebagai integer menit — belum ada grammar durasi terstruktur
(ISO 8601 atau pasangan nilai+satuan), penyederhanaan yang disengaja. `file` (CAP-F06) ✅ —
upload sungguhan tersimpan di disk (`prototype/go/uploads/`, key token 32-byte tak-tertebak),
disajikan di `GET /files/{key}`; `options.compress: true` menjalankan pipeline resize (`golang.
org/x/image/draw`) + re-encode JPEG/WebP sungguhan (`github.com/chai2010/webp` + `libwebp` host)
di server, terlepas dari kompresi client-side. `user` (CAP-F05) ✅ sudah diimplementasikan
sebelumnya — lihat baris CAP-F05 di `capability-registry.md`, bukan lagi teks bebas.

#### `child_table` (CAP-F16, ✅ diimplementasikan 2026-07-12) — pilihan storage, bukan pemetaan tipe field

Di `.menata`, "satu Journal Entry punya banyak Journal Entry Line" tidak pernah dituliskan sebagai
Field khusus — itu Object biasa (`Journal Entry Line`) dengan satu Field yang merujuk balik ke
induknya (`- Journal Entry : Journal Entry`), pola Object References standar
(`specification/001-object.md` §Relationships di repo `menata`), lihat
`prototype/go/docs/examples/accounting-journal-entry-line.menata`. (Koreksi 2026-07-11: revisi
sebelumnya menulis ini sebagai notasi sementara `Table of (...)` di induknya, mengira belum ada
grammar Language untuk pola ini — ternyata sudah ada, lihat `capability-registry.md` CAP-F16.)

Yang jadi tugas developer saat menerjemahkan ke Runtime Metadata: Machine hasil terjemahan Object
semacam ini **boleh** disimpan sebagai `child_table` (baris-baris record biasa dari machine anak,
di-scope ke satu induk) alih-alih machine independen biasa — itu murni keputusan storage/query di
level Machine Interpretation, tidak tercermin apa pun di `.menata`. Contoh: `Journal Entry Line`
(harus tetap bisa di-query lintas induk untuk laporan agregat — lihat catatan
"reporting-independence" di `capability-registry.md` CAP-F16) vs. `Item Unit Conversion` (murni
lookup per-induk, tidak pernah di-query lintas record) — keduanya ditulis dengan pola `.menata` yang
identik, tapi bisa berbeda strategi penyimpanan.

#### Money, User, File — "reference sugar" (Study 15)

`money`, `user`, dan `file` bukan tipe primitif berdiri sendiri secara konseptual — ketiganya adalah
**reference dengan target yang sudah ditentukan di depan**, dipertahankan sebagai tipe bernama
terpisah hanya karena runtime belum punya target-nya:

| Tipe | Target reference-nya | Status target |
|------|----------------------|----------------|
| `user` | Identitas platform (siapa penggunanya) | ✅ CAP-F05 — picker sungguhan ke-scope `user_application_roles`, menyimpan user id (bukan nama), referential integrity di-enforce |
| `money` | Currency (kode + kurs) | ⚠️ CAP-F08 sudah render/validasi/hitung dengan benar, tapi `currency`/`currency_field` masih sekadar field `value_list`/`text` biasa berisi kode — Currency sebagai Machine master-data sungguhan (CAP-F17) masih terbuka |
| `file` | Entitas File/Document terkelola runtime | ⚠️ CAP-F06 — upload tersimpan & tersaji nyata (lihat catatan status di atas), tapi masih field value berbasis disk, belum jadi Machine mandiri dengan identity/lifecycle sendiri |

Detail lengkap pohon keputusan + kalibrasi: `runtime/benchmarks/005-field-modeling-decision-framework.md`.

**Wajib untuk `type: money`:** sertakan `currency` di `options` (contoh: `{"currency":"IDR"}`), atau
`currency_field` kalau nilainya bisa beda per record (referensi ke field lain di machine yang sama).
Metadata `type: money` **tanpa** salah satu dari keduanya dianggap tidak lengkap — sama seperti
`value_list` tanpa `values` atau `reference` tanpa `target_machine`. **CAP-X05 (metadata
validation before load) sudah ✅ diimplementasikan (2026-07-12)** — server menolak boot untuk
operator constraint/condition yang tidak dikenal, dan untuk `owner_field` yang tidak menunjuk ke
Field `type: user`. Yang **belum** tercakup: validasi companion-field `money` (`currency`/
`currency_field`) — masih menunggu implementasi nyata `money` sebagai reference sugar (CAP-F17),
jadi untuk saat ini tetap pastikan manual saat menulis seed.

#### Field `required`

Field yang tidak disebutkan sebagai opsional di `.menata` diasumsikan `required = true`.  
Field `Status` adalah pengecualian — selalu `required = false` karena nilainya diset oleh Events.

#### Field Status — WAJIB didefinisikan manual

> ⚠️ Status field **tidak** dibuat otomatis oleh runtime. Harus didefinisikan secara eksplisit.

```sql
INSERT INTO fields (...) VALUES
    ('fld_lr_status', 'mch_leave_request', 'Status', 'value_list', 5, false,
        '{"values":["Draft","Submitted","Approved","Rejected","Cancelled"]}');
```

Nilai awal Status saat record dibuat = nilai pertama dalam array `values` (dalam contoh ini: `Draft`). Runtime menetapkan ini secara otomatis saat `Create`.

`position` untuk Status selalu diletakkan terakhir (nilai tertinggi).

---

### Events dan Actions

**.menata**
```
When Submit

    Status Submitted

When Approve

    Status Approved

    Notify Employee
```

**YAML**
```yaml
events:
  - id: evt_lr_submit
    name: Submit
    condition: { field: fld_lr_status, operator: equals, value: Draft }
    actions:
      - set_field: { field: fld_lr_status, value: Submitted }

  - id: evt_lr_approve
    name: Approve
    condition: { field: fld_lr_status, operator: equals, value: Submitted }
    actions:
      - set_field: { field: fld_lr_status, value: Approved }
      - set_field: { field: fld_lr_approved_date, value: today }
      - set_field: { field: fld_lr_approved_by, value: current_user }
      - notify: { role: Employee }
```

**SQL**
```sql
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_lr_submit',  'mch_leave_request', 'Submit',  0,
        '{"field":"fld_lr_status","operator":"equals","value":"Draft"}'),
    ('evt_lr_approve', 'mch_leave_request', 'Approve', 1,
        '{"field":"fld_lr_status","operator":"equals","value":"Submitted"}');

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_lr_submit',  'set_field', 0, '{"field":"fld_lr_status","value":"Submitted"}'),
    ('evt_lr_approve', 'set_field', 0, '{"field":"fld_lr_status","value":"Approved"}'),
    ('evt_lr_approve', 'set_field', 1, '{"field":"fld_lr_approved_date","value":"today"}'),
    ('evt_lr_approve', 'set_field', 2, '{"field":"fld_lr_approved_by","value":"current_user"}'),
    ('evt_lr_approve', 'notify',    3, '{"role":"Employee"}');
```

#### Event `condition` — guard (CAP-E06)

Sama bentuknya dengan `condition` di Constraint. Event hanya boleh dipicu kalau data record **saat
ini** (sebelum aksi event ini berjalan) memenuhi syarat ini — kalau tidak, runtime menolak trigger-nya
(400), tidak diam-diam diabaikan. Ini merealisasikan `if` yang sudah ada di grammar Event bahasa
Menata (`003-event.md` §Conditions di repo `menata`) — bukan konsep baru untuk penulis `.menata`,
runtime-nya saja yang baru bisa mengeksekusinya sejak CAP-E06. `condition = NULL` berarti Event boleh
dipicu dari state mana pun.

#### Tipe Action — pemetaan

| Di .menata | `type` di DB | `params` |
|------------|-------------|---------|
| `Status <Nilai>` | `set_field` | `{"field":"fld_*_status","value":"<Nilai>"}` |
| `Notify <Role>` | `notify` | `{"role":"<Role>"}` — atau `{"recipient_field":"fld_*"}` untuk penerima dinamis (CAP-A04, ✅) |
| `Record <Nama>` | `record` | `{"name":"<Nama>"}` |
| — | `create_record` | `{"machine":"mch_*","fields":{"<field_target>":"field:<fld_source>"}}` — CAP-A06, ✅ diimplementasikan 2026-07-12, membuat record baru di Machine lain. **Key-nya `machine`, BUKAN `target_machine`** (itu cuma dipakai field type `reference`) — dan nilai di `fields` butuh prefix `"field:"` untuk menyalin dari record sumber, kalau tidak dianggap literal |
| — | `cross_set_field` | mengubah field di record **lain** yang sudah ada, dijangkau lewat field `reference` — CAP-A13, ✅ |
| — | `batch_generate` | membuat N record sekaligus dari satu action, N dari field atau literal — CAP-A15, ✅ |
| — (lihat §Machine `config`) | `activate_next` | `{"mode_field":"fld_*_approval_mode"}` — CAP-A07 |
| — (lihat §Machine `config`) | `aggregate_status` | `{"parent_field":"fld_*_document","parent_event_if_all_approved":"evt_*","parent_event_if_any_rejected":"evt_*"}` — CAP-A08 |

Setiap action bisa dibungkus `if: {field, operator, value}` (CAP-A09, ✅) supaya hanya jalan kalau
kondisi itu benar — beda dari `condition` di level Event (yang menentukan boleh-tidaknya Event
dipicu sama sekali).

`position` di `event_actions` menentukan urutan eksekusi dalam satu Event. Dimulai dari 0.

#### `set_field` dengan nilai dinamis (CAP-A02, CAP-A11, CAP-A12, CAP-P04)

`value: today`, `value: now`, dan `value: current_user` di-resolve saat Event benar-benar dipicu,
bukan disimpan sebagai teks literal "today"/"now"/"current_user". `current_user` di-resolve ke
identitas sungguhan orang yang sedang bertindak (CAP-X02 — nama akun yang sudah terautentikasi,
bukan sekadar string role).

Sejak 2026-07-12 (Batch 3), `value` juga mendukung **aritmetika tanggal**: `"today + 7 Days"`,
`"today - 3 Days"`, `"fld_completed_at + 1 Month"` — basis-nya `today`/`now`/id Field bertipe
`date`, unit-nya `Day(s)`/`Week(s)`/`Month(s)`/`Year(s)`/`Business Day(s)` (CAP-A11). Varian
`Business Day(s)` juga melewati akhir pekan **dan** hari libur yang dideklarasikan Workspace-nya
(`workspace_holidays`, CAP-O06). `value: "next"` (CAP-A12) memajukan field `value_list` ke opsi
berikutnya sesuai urutan `values`-nya sendiri (tidak berputar balik setelah opsi terakhir).
`value: "input:<field_id>"` (CAP-P04) membaca dari input yang dikumpulkan segar saat Event
dipicu (lihat `input_fields` pada Event, bukan dari data record yang sudah ada) — pola ini
dipakai untuk delegasi/reassignment (form trigger mengirim nilai baru dalam satu request yang
sama dengan Event-nya).

#### `activate_next` + `aggregate_status` — workflow sequential/aggregate (CAP-A07, CAP-A08)

Dua action ini bekerja berpasangan untuk pola "Document dengan beberapa Approval Step", persis
seperti `approval-document.yaml` + `approval-step.yaml`:

```yaml
# di evt_as_approve (Approval Step)
actions:
  - set_field: { field: fld_as_decision, value: Approved }
  - activate_next: { mode_field: fld_ad_approval_mode }
  - aggregate_status:
      parent_field: fld_as_document
      parent_event_if_all_approved: evt_ad_approve
      parent_event_if_any_rejected: evt_ad_reject
```

- **`activate_next`** — kalau Machine parent-nya (dicari lewat `mode_field`) mode-nya `Sequential`,
  action ini mengirim notifikasi ke approver step berikutnya yang masih Pending. Kalau `Parallel`,
  tidak melakukan apa-apa.
- **Penegakan urutan sequential-nya sendiri bukan di action ini** — cukup deklarasikan
  `aggregate_status` pada Event yang sama, dan runtime otomatis menolak (400) kalau ada sibling step
  dengan urutan lebih awal yang masih Pending, sebelum Event ini sempat berjalan. Ini keputusan
  desain yang disengaja: WCP-1 Sequence (pola yang direalisasikan capability ini) memang didefinisikan
  lewat penegakan — kalau cuma notifikasi tanpa penegakan, mode Sequential dan Parallel akan
  berperilaku identik.
- **`aggregate_status`** — setelah Event ini commit, runtime mengecek semua sibling record dengan
  `parent_field` yang sama: kalau ada satu saja yang Rejected, `parent_event_if_any_rejected`
  langsung dipicu di parent (tidak menunggu sibling lain memutuskan). Kalau **semua** sudah Approved,
  baru `parent_event_if_all_approved` dipicu. Event di parent ini dipicu lewat jalur yang **persis
  sama** dengan Event yang dipicu manual lewat HTTP — `condition` (CAP-E06) dan Constraints
  (CAP-C09) parent tetap berlaku, jadi transisi yang dipicu sistem tidak bisa melewati pemeriksaan
  yang harus dilewati transisi yang dipicu manusia.
- Field parent/sequence/decision **tidak** disebut eksplisit di parameter action — runtime
  menemukannya lewat heuristik (field bertipe `reference` yang menunjuk ke parent; field yang
  namanya persis "Sequence"/"Decision"/"Approver"). Ini keterbatasan prototipe yang disengaja dan
  didokumentasikan, bukan sihir tersembunyi — bahasa Menata belum punya cara bagi penulis `.menata`
  untuk menamai field-field ini secara eksplisit. Detail lengkap: `capability-registry.md` (CAP-A07).

---

### Constraints

> Sejak CAP-C09, setiap Constraint yang dideklarasikan di sini dicek bukan cuma saat Create, tapi
> juga setiap kali Event dipicu — dievaluasi terhadap data record *setelah* aksi Event tersebut
> akan berjalan, sebelum benar-benar disimpan. Record yang valid saat dibuat bisa jadi tidak valid
> lagi saat sebuah Event terjadi (misal constraint berbasis tanggal, karena waktu berjalan) — tidak
> perlu ditulis dua kali, cukup satu Constraint yang sama otomatis berlaku di kedua momen.

**.menata**
```
Constraints

- Reason is required.
- Start Date must be after today.
- Attachment is required.

    if Design Type = Banner 2:1
```

**YAML**
```yaml
constraints:
  - id: cst_lr_reason_required
    rule: Reason is required.
    expression:
      field: fld_lr_reason
      operator: required

  - id: cst_lr_start_future
    rule: Start Date must be after today.
    expression:
      field: fld_lr_start_date
      operator: after
      value: today

  - id: cst_dr_attachment_for_banner
    rule: Attachment is required for Banner design type.
    expression:
      field: fld_attachment
      operator: required
    condition:
      field: fld_design_type
      operator: equals
      value: Banner 2:1
```

**SQL**
```sql
INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_lr_reason_required',
     'mch_leave_request',
     'Reason is required.',
     '{"field":"fld_lr_reason","operator":"required"}',
     NULL, 0),

    ('cst_lr_start_future',
     'mch_leave_request',
     'Start Date must be after today.',
     '{"field":"fld_lr_start_date","operator":"after","value":"today"}',
     NULL, 1),

    ('cst_dr_attachment_for_banner',
     'mch_design_request',
     'Attachment is required for Banner design type.',
     '{"field":"fld_attachment","operator":"required"}',
     '{"field":"fld_design_type","operator":"equals","value":"Banner 2:1"}',
     2);
```

#### Operator yang didukung runtime (per 2026-07-12, Batch 1)

| Kalimat di .menata | `operator` | `value` | Keterangan |
|--------------------|-----------|---------|-----------|
| `<Field> is required.` | `required` | — | Field tidak boleh kosong |
| `<Field> must be after today.` | `after` | `"today"` | Tanggal harus setelah hari ini |
| `<Field> must be after <Field2>.` | `after` | id Field lain | Perbandingan antar-field pada record yang sama — CAP-C07, ✅ |
| `<Field> must be greater than <N>.` | `greater_than` | nilai string | CAP-C05, ✅ |
| `<Field> must be less than <N>.` | `less_than` | nilai string | CAP-C05, ✅ |
| `<Field> = <Nilai>` *(di `if`)* | `equals` | nilai string | Untuk condition |
| `<Field> ≠ <Nilai>` *(di `if`)* | `not_equals` | nilai string | Untuk condition |

**Keunikan komposit** (CAP-C12, ✅) bukan operator, bentuknya beda — pakai kolom
`unique_together` (array id Field), bukan `expression`:

```sql
-- 'unique_together' berisi array field id, bukan {field, operator, value}
INSERT INTO constraints (id, machine_id, rule, unique_together, position) VALUES
    ('cst_jel_seq_unique', 'mch_journal_entry_line', 'Sequence must be unique within an Entry.',
     '["fld_jel_entry","fld_jel_sequence"]', 3);
```

Operator lain (misal "Amount must be at least zero" → `greater_than_or_equal`) valid sebagai
Business Knowledge di `.menata`, tapi masih belum diimplementasikan di runtime prototype —
**tidak error saat load, cuma diam-diam tidak pernah gagal** (`constraint.Eval` default-nya
menganggap "terpenuhi" untuk operator yang tidak dikenal). Tetap tuliskan di `.menata` — operator
baru ditambahkan ke runtime tanpa mengubah `.menata`, tapi jangan asumsikan sudah jalan di
runtime hanya karena loadnya tidak error.

`condition` = `NULL` untuk constraint tanpa kondisi.

---

### Permissions

**.menata**
```
Permissions

Employee

- Submit
- Cancel

Manager

- Approve
- Reject
```

**YAML**
```yaml
permissions:
  - role: Employee
    events: [ evt_lr_submit, evt_lr_cancel ]

  - role: Manager
    events: [ evt_lr_approve, evt_lr_reject ]
```

**SQL**
```sql
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_lr_employee', 'mch_leave_request', 'Employee',
        ARRAY['evt_lr_submit','evt_lr_cancel']),
    ('perm_lr_manager',  'mch_leave_request', 'Manager',
        ARRAY['evt_lr_approve','evt_lr_reject']);
```

Nilai `role` di sini harus **persis sama** dengan role yang di-assign ke user lewat
`/admin/users` (CAP-O01, `user_application_roles`) untuk Application tempat Machine ini berada
— bukan lagi cookie yang dikirim client sendiri (CAP-X02, cookie `menata_role` sudah tidak
ada). Case-sensitive.

**Kepemilikan record (`owner_field`) dan akses CRUD (`can_read`/`can_create`/`can_edit`)**

Dua penambahan (2026-07-12), independen dari `events`:

```yaml
permissions:
  - role: Approver
    events: [ evt_as_approve, evt_as_reject ]
    owner_field: fld_as_approver   # hanya identity yang namanya cocok dengan
                                    # nilai field ini di record yang boleh
                                    # bertindak -- bukan sekadar siapa saja
                                    # yang memegang role "Approver"

  - role: Submitter
    events: []
    can_read: true
    can_create: true
    can_edit: false                # masing-masing default true kalau tidak ditulis

  - role: Staff                    # CAP-P06, ✅ 2026-07-12 — visibilitas per-field
    events: []
    can_read: true
    hidden_fields: [ fld_salary ]  # field ini hilang total dari List/Detail/Form untuk role ini

  - role: Visitor                  # CAP-P07, ✅ 2026-07-12 — akses anonim (tanpa login)
    events: []
    can_read: true                 # hanya GET; POST selalu ditolak untuk request tanpa sesi apa pun
```

- `owner_field` (opsional, id Field di Machine yang sama): kalau diisi, `events`
  yang terdaftar butuh **identity** orang yang login (akun sungguhan, CAP-X02)
  sama persis dengan nilai field itu di record-nya, bukan cuma cocok role-nya.
- `can_read`/`can_create`/`can_edit` (opsional, default `true`): akses baca/
  buat/ubah ke record suatu Machine, terpisah dari Event apa saja yang boleh
  dipicu. **Deny-by-default per Machine** — role yang sama sekali tidak punya
  baris Permission di suatu Machine otomatis tidak punya akses baca/buat/ubah
  sama sekali, bukan diizinkan diam-diam seperti sebelumnya.
- `hidden_fields` (opsional, array id Field, CAP-P06): field yang dikecualikan dari
  List/Detail/Form untuk role ini — bukan cuma read-only, benar-benar tidak muncul.
  Role lain di Machine yang sama tanpa `hidden_fields` (atau daftar berbeda) tetap
  melihatnya seperti biasa; ini pengaturan per-role, bukan per-Machine.
- `Visitor` (CAP-P07) bukan kata kunci khusus — itu nama role biasa. Yang membuatnya
  "akses anonim" murni karena request tanpa sesi login di-resolve sebagai role
  `Visitor` — jadi baris Permission dengan `role: Visitor` dan `can_read: true` itulah
  yang benar-benar memberi akses baca publik. Machine tanpa baris `Visitor` menolak
  request anonim, sama seperti role lain (deny-by-default). Request anonim tidak
  pernah bisa menulis, berapa pun `can_create`/`can_edit` yang dideklarasikan.

**Siapa yang benar-benar memegang suatu `role` adalah urusan CAP-O01, bukan file ini.** `role`
di sini cuma **mendeklarasikan kosakatanya** — nama role apa saja yang ada dan masing-masing
boleh apa di Machine ini. Orang sungguhan di-assign satu role untuk satu Application secara
utuh (semua Machine di dalam satu Application berbagi kosakata role yang sama) lewat
`user_application_roles`, satu baris per pasangan `(user, application)`, diatur lewat halaman
`/admin/users` — bukan bagian dari Runtime Metadata, dan juga bukan pilihan bebas saat login:
role tidak lagi dideklarasikan sendiri (CAP-X02), tapi di-assign lebih dulu oleh seorang
Admin workspace. Orang yang sama bisa pegang role berbeda di Application berbeda pada saat
bersamaan (mis. "Requester" di sini, "Approver" di sana) tanpa langkah "ganti role" — role-nya
untuk suatu halaman resolve dari Application mana halaman itu berada.

**Tambahan (2026-08-23, CAP-O07):** `user_application_roles` bukan lagi satu-satunya jalur untuk
memegang suatu role — role juga bisa diberikan ke sebuah **Group** (`groups`/
`group_application_roles`), dengan orang ditambah/dikeluarkan dari keanggotaan Group itu, bukan
menyentuh baris masing-masing orang. Himpunan role *efektif* seseorang untuk satu Application
adalah gabungan dari assignment langsungnya (`user_application_roles`) dan setiap role yang
dipegang oleh Group manapun yang dia ikuti di situ — role lewat jalur mana pun memberi akses yang
sama persis; kosakata role yang dideklarasikan file ini tidak berubah karena asal assignment-nya.
Dikelola di tempat yang sama dengan jalur langsung, bagian Groups di `/admin/users`.

---

### Views

**.menata**
```
Views

- Leave Request Form : Form
- My Requests : List
- Pending Approvals : List
- Leave Request Detail : Detail
```

**YAML**
```yaml
views:
  - id: vw_lr_form
    name: Leave Request Form
    type: form
    fields:
      - fld_lr_employee
      - fld_lr_leave_type
      - fld_lr_start_date
      - fld_lr_end_date
      - fld_lr_reason

  - id: vw_lr_my_requests
    name: My Requests
    type: list
    columns:
      - fld_lr_leave_type
      - fld_lr_start_date
      - fld_lr_end_date
      - fld_lr_status
    default_sort:
      field: fld_lr_start_date
      direction: asc

  - id: vw_lr_pending
    name: Pending Approvals
    type: list
    columns:
      - fld_lr_employee
      - fld_lr_leave_type
      - fld_lr_start_date
      - fld_lr_status

  - id: vw_lr_detail
    name: Leave Request Detail
    type: detail
```

**SQL**
```sql
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_lr_form', 'mch_leave_request', 'Leave Request Form', 'form', 0,
        '{"fields":["fld_lr_employee","fld_lr_leave_type","fld_lr_start_date","fld_lr_end_date","fld_lr_reason"]}'),

    ('vw_lr_my_requests', 'mch_leave_request', 'My Requests', 'list', 1,
        '{"columns":["fld_lr_leave_type","fld_lr_start_date","fld_lr_end_date","fld_lr_status"],
          "default_sort":{"field":"fld_lr_start_date","direction":"asc"}}'),

    ('vw_lr_pending', 'mch_leave_request', 'Pending Approvals', 'list', 2,
        '{"columns":["fld_lr_employee","fld_lr_leave_type","fld_lr_start_date","fld_lr_status"]}'),

    ('vw_lr_detail', 'mch_leave_request', 'Leave Request Detail', 'detail', 3, '{}');
```

#### View `config` per tipe (per 2026-07-12)

| Tipe View | Kunci di `config` | Keterangan |
|-----------|------------------|-----------|
| `form` | `fields`, `child_lines` (CAP-F16 ✅), `steps` (CAP-V12 ✅) | Array field id yang tampil di form, berurutan. `child_lines` menyisipkan N baris Machine anak sekaligus dalam satu submit (dokumen header-detail). `steps` memecah `fields` jadi beberapa langkah wizard, tiap entry array-nya satu langkah |
| `list` | `columns`, `default_sort`, `filter` (CAP-V09/V05 ✅), `manual_order` (CAP-V14 ✅) | Kolom tabel; sort opsional. `filter` = daftar kondisi AND, bentuknya sama dengan `constraint.expression` (`$current_user` = sentinel "record milik saya", CAP-V05). `manual_order: true` mengaktifkan tombol Up/Down, urut berdasar kolom `sort_order`, bukan `default_sort` |
| `detail` | — | `{}` — runtime menampilkan semua field. Sub-list reverse-reference (CAP-V06 ✅ — record Machine lain yang field `reference`-nya menunjuk balik ke record ini) otomatis muncul, tanpa config |
| `calendar`, `timeline` | `columns`, `date_field` (CAP-V07 ✅) | Record dikelompokkan/diurutkan berdasar `date_field` |
| `dashboard` | `sections` (CAP-V10 ✅) | Array `{title, machine, group_field?}` — tiap section bisa dari Machine berbeda-beda, itu inti CAP-V10 dibanding `list`/`report` biasa yang cuma satu Machine |
| `report` | `report: {machine, group_field, sum_fields}` (CAP-V13 ✅) | Rollup/group-by atas record Machine LAIN, dihitung saat render, tidak disimpan (mis. Trial Balance) |

Pencarian bebas teks (`?q=`, CAP-V08 ✅) selalu tersedia di `list` tanpa config apa pun.

Field Status **tidak perlu** dimasukkan ke `config.fields` pada view Form — Status diset oleh Events, bukan oleh user input.

Contoh lengkap tiap bentuk `config` (worked examples): `runtime-metadata-schema.md` §"View `config` per type".

---

## Urutan INSERT yang Benar

Urutan penting karena foreign key constraints:

```
1. workspaces
2. applications      (→ workspaces)
3. machines          (→ applications)
4. fields            (→ machines)
5. events            (→ machines)
6. event_actions     (→ events)
7. constraints       (→ machines)
8. permissions       (→ machines)
9. views             (→ machines)
```

Gunakan `ON CONFLICT (id) DO NOTHING` agar seed aman dijalankan ulang.

---

## Yang Bikin Loader Gagal vs Yang Diam-diam Tidak Jalan

Ditemukan langsung (2026-07-12) saat mengonversi 50 file contoh `.yaml` yang sebelumnya
belum pernah benar-benar dimuat ke runtime — beberapa cara gagal yang tidak kelihatan cuma
dari baca grammar-nya saja. Berlaku sama baik metadata-nya ditulis manusia maupun AI.

**Satu Machine yang salah bikin SELURUH server gagal boot, bukan cuma Machine itu.**
`Loader.LoadAll` memuat seluruh pohon setiap Workspace dalam satu pass; satu
field/event/constraint/permission yang tidak valid di mana pun membatalkan seluruh proses
boot (`os.Exit(1)` di `cmd/server/main.go`) — tidak ada partial load, tidak ada karantina
per-Machine. Kesalahan metadata di satu Application yang bahkan tidak sedang Anda kerjakan
bisa mematikan seluruh runtime untuk semua workspace lain juga.

**`value` di `constraint.expression`, `constraint.condition`, atau `event.condition` HARUS
string, walau kelihatannya angka.** `value: 100` (angka YAML/JSON) bikin loader crash
(`cannot unmarshal number into Go struct field ConstraintExpression.value of type string`)
— tulis `value: "100"`. Ini gagal fatal saat load, bukan diam-diam tidak jalan.

**Operator yang jalan per 2026-07-12 (Batch 1)**: `required`, `equals`, `not_equals`, `after`
(literal `"today"` saja), `greater_than`, `less_than` (CAP-C05), plus **perbandingan
antar-field** — `value` boleh berisi id Field lain, bukan literal, mis. `{field: fld_end_date,
operator: after, value: fld_start_date}` (CAP-C07). **Keunikan komposit** (CAP-C12) bentuknya
beda sama sekali, bukan operator pada satu Constraint — lihat `unique_together` di §Constraints
atas. Masih **belum didukung, masih diam-diam tidak pernah aktif** (default case
`constraint.Eval` tetap mengembalikan `true` untuk operator tak dikenal): `greater_than_or_equal`,
`less_than_or_equal`, `unique` sebagai operator field-level biasa (pakai `unique_together`),
dan bentuk agregat (`aggregate: sum`) atau `conditions:` jamak. Jangan ditulis seolah-olah
jalan — sebutkan gap-nya saja (baris CAP-C10 di `capability-registry.md`).

**`set_field.value` mendukung literal string, tiga token dinamis (`today`, `now`,
`current_user`, CAP-A02), referensi `"input:<field_id>"` (CAP-P04), dan — per 2026-07-12
(Batch 3) — aritmetika tanggal**: `"<basis> + N <unit>"` / `"<basis> - N <unit>"`, basis-nya
`today`/`now`/id Field, unit-nya `Day(s)`/`Week(s)`/`Month(s)`/`Year(s)`/`Business Day(s)`
(CAP-A11; varian `Business Day(s)` juga melewati akhir pekan dan hari libur Workspace,
CAP-O06). `value: "next"` (CAP-A12) memajukan field `value_list` ke opsi berikutnya. Selain
itu — pemanggilan fungsi (`raise_one_level(priority)`, `sla_offset(priority)`), aritmatika
field non-tanggal (`reopen_count + 1`), interpolasi template (`{{ this.field }}`), pembacaan
`previous(field)` — **sama sekali tidak dievaluasi**. Nilainya ditulis apa adanya ke record,
verbatim, data yang diam-diam salah, bukan error.

**`create_record` sudah diimplementasikan (CAP-A06, ✅, 2026-07-12)** — membuat record
sungguhan di Machine lain, memetakan/menyalin field dari record sumber. Dua tipe action lain
di luar tabel awal juga sudah ada: `cross_set_field` (CAP-A13, mengubah field di record LAIN
yang sudah ada, dijangkau lewat field `reference`) dan `batch_generate` (CAP-A15, membuat N
record dari satu action). Action apa pun boleh dibungkus `if: {field, operator, value}`
(CAP-A09) supaya kondisional dalam satu Event yang punya beberapa action.

**Sumber trigger Event bukan cuma HTTP POST dan Event lain.** `schedule: {time: "HH:MM"}`
(CAP-E02) atau `schedule: {date_field, offset_days}` (CAP-E03) membuat Event dipicu otomatis
oleh scheduler background (real, `time.Ticker` sekali semenit, bukan simulasi) — dua kunci ini
saling eksklusif. Webhook eksternal (CAP-E04) tidak butuh kunci di Event sama sekali; Machine-
nya yang deklarasikan `config.webhook_secret`, lalu `POST /webhooks/{machine}/{record}/{event}`
dengan header `X-Webhook-Secret` yang cocok memicunya — tanpa sesi login, tanpa CSRF.

**`target_machine` field `reference` harus Machine id asli yang benar-benar ada di load yang
sama** — termasuk target reserved/pseudo seperti `"$identity"` (flavor (b) CAP-F13 yang
belum diimplementasikan, target identitas bawaan) dianggap dangling dan gagal load, blast
radius sama seperti di atas. Kalau maksudnya "orang yang sedang login," itu `type: user`
(CAP-F05), bukan `reference` dengan target karangan.

**`permissions.owner_field` harus menunjuk Field yang dideklarasikan `type: user`** di
Machine yang sama (CAP-P02/CAP-F05, ditegakkan saat load sejak 2026-07-12) — menunjuk ke
field `text` atau tipe lain bikin load gagal. Hilangkan `owner_field` sama sekali untuk
gating berbasis role saja kalau memang tidak ada Field yang benar-benar mewakili "orang
spesifik yang harus bertindak."

**Key YAML/JSON yang tidak dikenali diam-diam diabaikan, bukan ditolak.** Decoding JSON
bawaan Go mengabaikan field yang tidak dideklarasikan struct-nya — blok `views.filter`
(CAP-V09, belum diimplementasikan) tidak error, cuma hilang tanpa jejak. Tidak adanya error
saat load BUKAN konfirmasi bahwa semua yang ditulis dipahami runtime; selalu silang-cek ke
dokumen ini dan `capability-registry.md` soal apa yang benar-benar sudah diimplementasikan,
bukan cuma "apakah tadi berhasil dimuat."

**Kalau Machine satu case tersebar di beberapa file yang berbagi satu Workspace/Application**
(pola umum begitu satu Application punya banyak Machine, satu file per Machine), tepat satu
dari file-file itu harus mendeklarasikan `workspace:`/`application:` sebagai objek penuh
(`{id, name, workspace: ws_id}`); file lain boleh mereferensikan Application-nya cukup
sebagai string id (`application: app_foo`) tanpa mengulang deklarasi penuh. Tidak ada yang
menegakkan bahwa *ada* file dalam grup itu yang mendeklarasikannya penuh — kalau tidak ada
sama sekali, Application-nya tidak pernah benar-benar dibuat dan setiap Machine yang
mereferensikannya lewat string jadi dangling.

---

## Checklist Sebelum Menjalankan Seed

- [ ] Semua ID mengikuti konvensi prefix
- [ ] Tidak ada ID yang duplikat dengan machine lain
- [ ] Field `position` berurutan dari 0
- [ ] Field Status didefinisikan manual dengan `required = false`
- [ ] Nilai pertama di `options.values` Status = nilai awal (biasanya `Draft`)
- [ ] Setiap Event punya minimal satu `set_field` action ke Status
- [ ] `event_actions.position` berurutan dalam satu Event
- [ ] Setiap Event yang ada di `.menata` tercantum di Permissions
- [ ] View Form tidak menyertakan field Status di `config.fields`
- [ ] View Detail menggunakan `config = '{}'`
- [ ] Field `reference` pakai key `target_machine` (bukan `machine`), menunjuk ke machine yang benar-benar ada
- [ ] Field `money` punya `currency` atau `currency_field` di `options` — jangan biarkan kosong
- [ ] Sudah cek `005-field-modeling-decision-framework.md` kalau ragu antara `value_list` vs `reference` vs primitif
- [ ] Event yang cuma boleh dipicu dari state tertentu punya `condition` (CAP-E06) — jangan andalkan disiplin pengguna
- [ ] Kalau pakai `aggregate_status`, machine child-nya punya field `Sequence`/`Decision`/`Approver` (nama persis, case-insensitive) kalau memang butuh gating sequential (CAP-A07)
- [ ] Kalau pakai workflow sequential/aggregate, parent machine-nya punya `config` (`approval_mode_field`, `steps_machine`, `steps_parent_field`) — lihat §Machine `config`
- [ ] Setiap `value` di `expression`/`condition` ditulis sebagai string (`"100"`, bukan `100`) — angka mentah bikin loader crash
- [ ] Operator di `constraint`/`condition` salah satu dari: `required`, `equals`, `not_equals`, `after`, `greater_than`, `less_than` (per 2026-07-12) — selain itu diam-diam tidak pernah aktif, bukan error
- [ ] `set_field.value` literal, `today`/`now`/`current_user`/`next`, `"input:<field>"`, atau aritmetika tanggal (`"today + N Days/Business Days"`) — bukan pemanggilan fungsi atau template
- [ ] `create_record`/`batch_generate` pakai key `machine` (BUKAN `target_machine` — itu cuma untuk field type `reference`), dan nilai `fields` yang menyalin dari record sumber butuh prefix `"field:"`
- [ ] `owner_field` di Permissions menunjuk Field yang `type: user` — bukan `text` atau tipe lain
- [ ] Kalau case-nya tersebar di beberapa file berbagi satu Application, tepat satu file mendeklarasikan `workspace:`/`application:` penuh

---

## API JSON otomatis dan Ekspor Metadata (CAP-X07/X08, 2026-07-12)

Bukan sesuatu yang ditulis penulis metadata — konsekuensi otomatis dari mendeklarasikan
sebuah Machine. Setiap Machine otomatis dapat `GET /api/{machine}`, `GET
/api/{machine}/{record}`, `POST /api/{machine}` (auth sesi + permission trimming yang sama
dengan route HTML; CSRF via header `X-CSRF-Token` karena body JSON tidak punya field form
`csrf_token`). Belum bisa lewat API ini: baris `child_table` (CAP-F16), langkah wizard
(CAP-V12), trigger event.

Admin workspace bisa ambil `GET /apps/{application}/export` — seluruh pohon metadata
Application itu (Field/Event/Constraint/Permission/View tiap Machine-nya) sebagai JSON,
langsung dari model yang sudah dimuat runtime. Read-only untuk saat ini — belum ada endpoint
import; metadata tetap hanya masuk lewat SQL seed (atau, di hulu-nya, pipeline authoring
`.menata` → Runtime Metadata yang dijelaskan seluruh dokumen ini).

**Perubahan semantik kegagalan (CAP-X12, 2026-07-12)**: kalau `create_record`/
`cross_set_field`/`batch_generate` gagal saat runtime (`machine` menunjuk id yang tidak
nyata, `record_field` menunjuk record yang sudah tidak ada), **SELURUH Event gagal dan TIDAK
ADA yang commit** — bukan cuma action itu yang dilewati. Setiap request HTTP berjalan dalam
satu transaksi database sungguhan; kegagalan satu action membatalkan semuanya, termasuk
`set_field` record itu sendiri dan action lain dalam Event yang sama yang tadinya berhasil
sendiri. Pastikan `machine`/`record_field` benar — salah ketik sekarang membatalkan seluruh
Event saat runtime, bukan cuma diam-diam tidak melakukan apa-apa seperti sebelumnya.

---

## Process Overlay — jalan pintas menulis proses (CAP-W01/W03/W04/W05, 2026-08-22)

Ada cara kedua menulis proses berbasis-state selain menulis `events`/`permissions`/Field Status
satu per satu seperti contoh Leave Request di atas: blok `process` pada Machine. Loader
**meng-compile**-nya menjadi persis Events/Permissions/Field yang sama seperti kalau ditulis
tangan — jadi hasil akhirnya identik, ini murni jalan pintas menulis, bukan mekanisme baru di
runtime. Referensi lengkap grammar-nya (compile mapping, aturan validasi, contoh penuh): bagian
"Process Overlay" di `../runtime-metadata-schema.md`.

```yaml
machine:
  id: mch_corrective_action
  process:
    states: [Open, Assigned, Submitted, Closed]
    transitions:
      - { name: Assign, from: Open, to: Assigned, actor: { role: Supervisor } }
      - { name: Submit, from: Assigned, to: Submitted, actor: { role: Worker },
          requirements: [ { type: evidence, target: mch_ca_photo, cardinality: "2..*" } ] }
      - { name: Close, from: Submitted, to: Closed, actor: { role: Supervisor } }
```

**Kapan pakai yang mana:** kalau prosesnya murni state-guard (kondisi `equals` pada satu Field
Status, actor role/owner_field) — pakai `process`, jauh lebih ringkas, dan tervalidasi sebagai
graph (state tak terjangkau, transition menggantung terdeteksi saat load, bukan saat runtime).
Kalau butuh kondisi yang lebih kaya (multi-field, `AggregateCondition`, `Schedule`,
`InputFields` untuk delegasi, dan lain-lain di luar cakupan `process`) — tulis `events` tangan
seperti biasa. **Satu Machine tidak boleh mendeklarasikan keduanya** — loader menolak load kalau
`process` dan `events` sama-sama ada, supaya tidak ada penggabungan ambigu yang ditebak diam-diam.

`requirements` mendukung dua tipe: `type: evidence` (jumlah record child Machine yang
mereferensikan balik ke Machine ini, lewat Field `reference`) dan, **sejak 2026-08-22**,
`type: approval` (CAP-W03 — lihat contoh di bawah). Empat tipe lain versi BRD pembanding (form,
entity, task, document, decision) masih belum ada padanannya, jangan ditulis seolah jalan.

**`type: approval` (kuorum deklaratif, CAP-W03)** — cara mendeklarasikan pola "N dari M harus
Approve" tanpa menulis `aggregate_status` tangan di Machine anak:

```yaml
process:
  transitions:
    - name: Approve
      from: Review
      to: Verified
      actor: { role: System }   # wajib System -- hasil kuorum tidak boleh bisa dipicu manusia
    - name: Reject
      from: Review
      to: Rejected
      actor: { role: System }
  requirements:
    - type: approval
      target: mch_review_vote       # Machine anak, satu record per pemilih
      min_approvals: 2              # "N" -- "M" tidak dideklarasikan, otomatis = jumlah
                                     # record anak yang ada saat itu
      on_quorum_approved: Approve   # nama transition di process Machine INI
      on_quorum_rejected: Reject    # keduanya WAJIB actor: { role: System }
```

`target` (`mch_review_vote` di atas) cukup Machine biasa, tak perlu punya `process` sendiri —
syaratnya cuma Field `value_list` bernama persis `"Decision"` (nilainya harus mencakup
`"Approved"`/`"Rejected"`) dan Field `reference` balik ke Machine yang mendeklarasikan
`requirements`. Detail lengkap aturan validasi: bagian "Process Overlay" di
`../runtime-metadata-schema.md`.

**`sla[]` (deadline per state, CAP-W04)** — opsional, sejajar dengan `transitions`/`auto` di
dalam blok `process` yang sama:

```yaml
process:
  sla:
    - state: Review
      duration: "2 Business Days"        # kosakata unit CAP-A11 (Day(s)/Week(s)/Month(s)/
                                          # Year(s)/Business Day(s)), bukan grammar baru
      on_breach:
        notify: { role: Manager }
        escalate_to: Escalated           # opsional -- boleh cuma notify tanpa pindah state
```

**`change_policy` pada Constraint (evolusi metadata effective-dated, CAP-W07)** — bukan bagian
dari `process`, tapi pasangan alaminya: menjawab pertanyaan "record yang sedang berjalan ikut
aturan baru ini atau tidak?" tanpa version-pinning:

```yaml
constraints:
  - id: cst_approval_ref_2026_policy
    rule: Approval Reference wajib untuk kasus yang dibuka di bawah kebijakan 2026.
    expression: { field: fld_approval_ref, operator: required }
    change_policy:
      applies_to: new_records          # atau records_in_states: [Draft, ...]
      effective_from: "2026-01-01"     # hanya untuk new_records
```

Tanpa `change_policy` sama sekali = perilaku hari ini (aturan langsung berlaku ke semua record,
`all_records`, eksplisit lewat ketiadaannya). Sebuah Constraint tidak boleh punya `change_policy`
DAN `condition` sekaligus (loader menolak, bukan menebak AND-list). Perubahan metadata dengan
`change_policy` baru terasa lewat CAP-X04's `POST /admin/reload` — tidak perlu restart server.

---

## Referensi

- [`menata-id/menata` guides](https://github.com/menata-id/menata/tree/main/guides) — cara menulis
  `.menata` (langkah sebelum ini; repo terpisah, khusus layer bahasa bisnis)
- `runtime/runtime-metadata-schema.md` — schema lengkap
- `runtime/benchmarks/005-field-modeling-decision-framework.md` — cara memilih tipe field yang tepat (primitif vs `value_list` vs `reference`), termasuk kenapa `money`/`user`/`file` adalah reference sugar
- `prototype/go/docs/examples/` — contoh lengkap Design Request dan Leave Request
- `prototype/go/docs/decisions/002-metadata-loading.md` — kapan restart diperlukan
