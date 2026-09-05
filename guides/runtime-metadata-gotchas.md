# Gotcha & Checklist Menulis Runtime Metadata

Dipecah dari [`writing-runtime-metadata.md`](writing-runtime-metadata.md) (2026-09-05, pure move —
tidak ada isi yang diubah, cuma dipindah) karena sifatnya beda dari grammar inti di file itu: ini
**log yang terus bertambah**, bukan referensi yang stabil — tiap kali ada bug/gotcha baru
ditemukan saat menulis atau menjalankan seed, catatannya ditambahkan di sini (append, don't
rewrite, sama seperti konvensi `roadmap.md`/`capability-registry.md`), bukan di file grammar
utama.

**Prasyarat:** sudah paham grammar dasar di `writing-runtime-metadata.md` — file ini adalah
"apa yang bisa salah" setelah tahu "bagaimana caranya benar", bukan pengganti file itu.

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
beda sama sekali, bukan operator pada satu Constraint — lihat `unique_together` di
`writing-runtime-metadata.md` §Constraints. Masih **belum didukung, masih diam-diam tidak pernah
aktif** (default case `constraint.Eval` tetap mengembalikan `true` untuk operator tak dikenal):
`greater_than_or_equal`, `less_than_or_equal`, `unique` sebagai operator field-level biasa (pakai
`unique_together`), dan bentuk agregat (`aggregate: sum`) atau `conditions:` jamak. Jangan
ditulis seolah-olah jalan — sebutkan gap-nya saja (baris CAP-C10 di `capability-registry.md`).

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
- [ ] Kalau pakai workflow sequential/aggregate, parent machine-nya punya `config` (`approval_mode_field`, `steps_machine`, `steps_parent_field`) — lihat §Machine `config` di `writing-runtime-metadata.md`
- [ ] Setiap `value` di `expression`/`condition` ditulis sebagai string (`"100"`, bukan `100`) — angka mentah bikin loader crash
- [ ] Operator di `constraint`/`condition` salah satu dari: `required`, `equals`, `not_equals`, `after`, `greater_than`, `less_than` (per 2026-07-12) — selain itu diam-diam tidak pernah aktif, bukan error
- [ ] `set_field.value` literal, `today`/`now`/`current_user`/`next`, `"input:<field>"`, atau aritmetika tanggal (`"today + N Days/Business Days"`) — bukan pemanggilan fungsi atau template
- [ ] `create_record`/`batch_generate` pakai key `machine` (BUKAN `target_machine` — itu cuma untuk field type `reference`), dan nilai `fields` yang menyalin dari record sumber butuh prefix `"field:"`
- [ ] `owner_field` di Permissions menunjuk Field yang `type: user` — bukan `text` atau tipe lain
- [ ] Kalau case-nya tersebar di beberapa file berbagi satu Application, tepat satu file mendeklarasikan `workspace:`/`application:` penuh
- [ ] **Blok `event_actions` di file seed ini belum pernah dijalankan sebelumnya terhadap database yang sedang dituju** — beda dari `fields`/`views`/`constraints`/`permissions`/`machines`, `event_actions` TIDAK punya `ON CONFLICT` (tidak ada natural key), jadi menjalankan ulang file yang sama menduplikasi setiap baris. Ditemukan langsung 2026-09-05: `event_actions` Case 3 terduplikasi 8× di database dev bersama, membuat setiap `notify`/`aggregate_status` asli terpicu 8 kali per satu Approve/Reject sungguhan. Cek dulu (`SELECT event_id, type, position, count(*) FROM event_actions GROUP BY 1,2,3 HAVING count(*) > 1`) sebelum menjalankan seed yang sudah punya event dengan action, terutama di database bersama/persisten yang mungkin sudah pernah menerima file ini
