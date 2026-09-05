# Menulis Process Overlay

Dipecah dari [`writing-runtime-metadata.md`](writing-runtime-metadata.md) (2026-09-05, pure move
— tidak ada isi yang diubah, cuma dipindah) karena ini mekanisme mandiri: sebuah jalan pintas
menulis, bukan bagian dari grammar Field/Event/Constraint/Permission/View inti.

**Prasyarat:** sudah paham grammar dasar (Fields, Events dan Actions, Permissions) di
`writing-runtime-metadata.md` — `process` di bawah ini cuma cara lain menulis Events/Permissions/
Field Status yang sama, bukan konsep baru untuk dipelajari dari nol.

---

## Process Overlay — jalan pintas menulis proses (CAP-W01/W03/W04/W05, 2026-08-22)

Ada cara kedua menulis proses berbasis-state selain menulis `events`/`permissions`/Field Status
satu per satu seperti contoh Leave Request di `writing-runtime-metadata.md`: blok `process` pada
Machine. Loader **meng-compile**-nya menjadi persis Events/Permissions/Field yang sama seperti
kalau ditulis tangan — jadi hasil akhirnya identik, ini murni jalan pintas menulis, bukan
mekanisme baru di runtime. Referensi lengkap grammar-nya (compile mapping, aturan validasi,
contoh penuh): bagian "Process Overlay" di `../runtime-metadata-schema.md`.

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
