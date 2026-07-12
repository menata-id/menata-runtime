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
berperilaku" (CAP-X03). Sejauh ini satu-satunya pemakai: workflow sequential/aggregate (CAP-A07/A08,
lihat §Events dan Actions di bawah):

```sql
UPDATE machines SET config = '{
  "approval_mode_field": "fld_ad_approval_mode",
  "steps_machine": "mch_approval_step",
  "steps_parent_field": "fld_as_document"
}' WHERE id = 'mch_approval_document';
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

> ⚠️ **Key yang benar untuk `reference` adalah `target_machine`, bukan `machine`.** Konsisten dengan
> contoh nyata di `prototype/go/docs/examples/approval-step.yaml` (`Document` field, `mch_approval_document`).

#### `child_table` (CAP-F16, ❌ belum diimplementasikan) — pilihan storage, bukan pemetaan tipe field

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
| `user` | Identitas platform (siapa penggunanya) | CAP-O01 (identity & role registry) sudah ✅ — `users`/`user_application_roles` — tapi `type: user` belum dimigrasikan untuk menunjuk ke situ sebagai target `reference`; masih dirender sebagai teks bebas (CAP-F05 ⚠️) |
| `money` | Currency (kode + kurs) | Menunggu CAP-O02 (master data designation) |
| `file` | Entitas File/Document terkelola runtime | Belum ada — CAP-F06 masih ⚠️ partial |

Detail lengkap pohon keputusan + kalibrasi: `runtime/benchmarks/005-field-modeling-decision-framework.md`.

**Wajib untuk `type: money`:** sertakan `currency` di `options` (contoh: `{"currency":"IDR"}`), atau
`currency_field` kalau nilainya bisa beda per record (referensi ke field lain di machine yang sama).
Metadata `type: money` **tanpa** salah satu dari keduanya dianggap tidak lengkap — sama seperti
`value_list` tanpa `values` atau `reference` tanpa `target_machine`. Ini dicegah CAP-X05 (metadata
validation before load) begitu diimplementasikan; untuk saat ini pastikan manual saat menulis seed.

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
| `Notify <Role>` | `notify` | `{"role":"<Role>"}` |
| `Record <Nama>` | `record` | `{"name":"<Nama>"}` |
| — (lihat §Machine `config`) | `activate_next` | `{"mode_field":"fld_*_approval_mode"}` — CAP-A07 |
| — (lihat §Machine `config`) | `aggregate_status` | `{"parent_field":"fld_*_document","parent_event_if_all_approved":"evt_*","parent_event_if_any_rejected":"evt_*"}` — CAP-A08 |

`position` di `event_actions` menentukan urutan eksekusi dalam satu Event. Dimulai dari 0.

#### `set_field` dengan nilai dinamis (CAP-A02)

`value: today`, `value: now`, dan `value: current_user` di-resolve saat Event benar-benar dipicu,
bukan disimpan sebagai teks literal "today"/"now"/"current_user". `current_user` di-resolve ke
identitas sungguhan orang yang sedang bertindak (CAP-X02 — nama akun yang sudah terautentikasi,
bukan sekadar string role).

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

#### Operator yang didukung runtime (prototype saat ini)

| Kalimat di .menata | `operator` | `value` | Keterangan |
|--------------------|-----------|---------|-----------|
| `<Field> is required.` | `required` | — | Field tidak boleh kosong |
| `<Field> must be after today.` | `after` | `"today"` | Tanggal harus setelah hari ini |
| `<Field> = <Nilai>` *(di `if`)* | `equals` | nilai string | Untuk condition |
| `<Field> ≠ <Nilai>` *(di `if`)* | `not_equals` | nilai string | Untuk condition |

Constraint dengan kalimat lain (misal "Amount must be greater than zero") valid sebagai Business Knowledge di `.menata`, tapi belum diimplementasikan di runtime prototype. Tetap tuliskan di `.menata` — operator baru ditambahkan ke runtime tanpa mengubah `.menata`.

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
```

- `owner_field` (opsional, id Field di Machine yang sama): kalau diisi, `events`
  yang terdaftar butuh **identity** orang yang login (akun sungguhan, CAP-X02)
  sama persis dengan nilai field itu di record-nya, bukan cuma cocok role-nya.
- `can_read`/`can_create`/`can_edit` (opsional, default `true`): akses baca/
  buat/ubah ke record suatu Machine, terpisah dari Event apa saja yang boleh
  dipicu. **Deny-by-default per Machine** — role yang sama sekali tidak punya
  baris Permission di suatu Machine otomatis tidak punya akses baca/buat/ubah
  sama sekali, bukan diizinkan diam-diam seperti sebelumnya.

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

#### View `config` per tipe

| Tipe View | Kunci di `config` | Keterangan |
|-----------|------------------|-----------|
| `form` | `fields` | Array field ID yang tampil di form, berurutan |
| `list` | `columns`, `default_sort` | Kolom tabel; sort opsional |
| `detail` | — | `{}` — runtime menampilkan semua field |

Field Status **tidak perlu** dimasukkan ke `config.fields` pada view Form — Status diset oleh Events, bukan oleh user input.

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

**Cuma empat operator constraint/condition yang benar-benar jalan**: `required`, `equals`,
`not_equals`, `after` (dan `after` cuma terhadap literal `"today"`). `before`,
`greater_than`, `less_than`, `greater_than_or_equal`, `unique`, atau bentuk
majemuk/agregat (`aggregate: sum`, `conditions:` jamak alih-alih `condition:` tunggal)
**bukan error — cuma diam-diam tidak pernah aktif** (default case `constraint.Eval`
mengembalikan `true`, artinya "terpenuhi", untuk operator apa pun yang tidak dikenali).
Constraint atau guard event yang pakai salah satu ini kelihatan terdeklarasi benar, dimuat
tanpa keluhan, lalu sama sekali tidak pernah melakukan apa-apa. Kalau butuh salah satu ini,
artinya belum didukung — jangan ditulis seolah-olah didukung; sebutkan gap-nya saja (lihat
baris CAP-C10/CAP-A09/CAP-C12 di `capability-registry.md`).

**`set_field.value` cuma mendukung literal string, atau salah satu dari tiga token
dinamis**: `today`, `now`, `current_user`. Selain itu — pemanggilan fungsi
(`raise_one_level(priority)`, `sla_offset(priority)`), aritmatika field
(`reopen_count + 1`), interpolasi template (`{{ this.field }}`), pembacaan
`previous(field)`, target dinamis `role:X` — **sama sekali tidak dievaluasi**. Nilainya
ditulis apa adanya ke record, verbatim, data yang diam-diam salah, bukan error.

**`create_record` terdeklarasi sebagai tipe action tapi belum ada implementasinya** —
`Executor.Persist` cuma log dan tidak melakukan apa-apa lagi (CAP-A06, ❌). Metadata yang
menyebutnya tetap dimuat dan jalan tanpa error; cuma tidak pernah benar-benar membuat
record yang dimaksud.

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
- [ ] Operator di `constraint`/`condition` cuma salah satu dari: `required`, `equals`, `not_equals`, `after` — selain itu diam-diam tidak pernah aktif, bukan error
- [ ] `set_field.value` cuma literal atau `today`/`now`/`current_user` — bukan pemanggilan fungsi, aritmatika, atau template
- [ ] `owner_field` di Permissions menunjuk Field yang `type: user` — bukan `text` atau tipe lain
- [ ] Kalau case-nya tersebar di beberapa file berbagi satu Application, tepat satu file mendeklarasikan `workspace:`/`application:` penuh

---

## Referensi

- [`menata-id/menata` guides](https://github.com/menata-id/menata/tree/main/guides) — cara menulis
  `.menata` (langkah sebelum ini; repo terpisah, khusus layer bahasa bisnis)
- `runtime/runtime-metadata-schema.md` — schema lengkap
- `runtime/benchmarks/005-field-modeling-decision-framework.md` — cara memilih tipe field yang tepat (primitif vs `value_list` vs `reference`), termasuk kenapa `money`/`user`/`file` adalah reference sugar
- `prototype/go/docs/examples/` — contoh lengkap Design Request dan Leave Request
- `prototype/go/docs/decisions/002-metadata-loading.md` — kapan restart diperlukan
