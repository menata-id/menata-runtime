# Panduan Menulis Runtime Metadata

Panduan ini menjelaskan cara menerjemahkan file `.menata` menjadi Runtime Metadata — dari YAML sampai ke SQL yang siap dijalankan.

**Untuk siapa:** Developer yang mengimplementasikan machine baru ke Menata Runtime.  
**Prasyarat:** Sudah ada file `.menata` yang ditulis domain expert (lihat `guides/writing-menata.md`).

Referensi schema lengkap: `runtime/runtime-metadata-schema.md`

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
| `Money` | `money` | `{}` |
| `Boolean` | `boolean` | `{}` |
| `Date` | `date` | `{}` |
| `Time` | `time` | `{}` |
| `Date Time` | `date_time` | `{}` |
| `Duration` | `duration` | `{}` |
| `User` | `user` | `{}` |
| `File` | `file` | `{}` |
| `A \| B \| C` | `value_list` | `{"values":["A","B","C"]}` |
| `Reference: X` | `reference` | `{"machine":"mch_x"}` |

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
    actions:
      - set_field: { field: fld_lr_status, value: Submitted }

  - id: evt_lr_approve
    name: Approve
    actions:
      - set_field: { field: fld_lr_status, value: Approved }
      - notify: { role: Employee }
```

**SQL**
```sql
INSERT INTO events (id, machine_id, name, position) VALUES
    ('evt_lr_submit',  'mch_leave_request', 'Submit',  0),
    ('evt_lr_approve', 'mch_leave_request', 'Approve', 1);

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_lr_submit',  'set_field', 0, '{"field":"fld_lr_status","value":"Submitted"}'),
    ('evt_lr_approve', 'set_field', 0, '{"field":"fld_lr_status","value":"Approved"}'),
    ('evt_lr_approve', 'notify',    1, '{"role":"Employee"}');
```

#### Tipe Action — pemetaan

| Di .menata | `type` di DB | `params` |
|------------|-------------|---------|
| `Status <Nilai>` | `set_field` | `{"field":"fld_*_status","value":"<Nilai>"}` |
| `Notify <Role>` | `notify` | `{"role":"<Role>"}` |
| `Record <Nama>` | `record` | `{"name":"<Nama>"}` |

`position` di `event_actions` menentukan urutan eksekusi dalam satu Event. Dimulai dari 0.

---

### Constraints

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

Nilai `role` di sini harus **persis sama** dengan cookie `menata_role` yang dikirim client. Case-sensitive.

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

---

## Referensi

- `guides/writing-menata.md` — cara menulis `.menata` (langkah sebelum ini)
- `runtime/runtime-metadata-schema.md` — schema lengkap
- `prototype/go/docs/examples/` — contoh lengkap Design Request dan Leave Request
- `prototype/go/docs/decisions/002-metadata-loading.md` — kapan restart diperlukan
