# Memecah Tampilan UI Menjadi Komponen yang Diatur Metadata

Panduan untuk AI agent (atau siapa pun) yang mendapat satu mockup/desain UI baru dan ingin tahu
cara memecahnya jadi komponen `templ` yang generik, dikendalikan oleh Runtime Metadata — bukan
kode Go yang menebak-nebak atau meng-hardcode satu use case tertentu.

**Untuk siapa:** developer/AI yang menambah View type baru, mekanisme komposisi View baru, atau
sekadar mengekstrak komponen presentasi dari mockup `app/web/static/ui-sample/*.html`.
**Prasyarat:** paham hierarki Runtime Metadata (`writing-runtime-metadata.md`) dan
`capability-lifecycle.md` §2 (uji admisi A1–A5).

Seluruh langkah di bawah ini bukan teori — ini rekonstruksi dari kesalahan nyata yang terjadi dan
dikoreksi dalam satu sesi kerja (2026-09-07), saat memecah `document-approval.html` dkk. jadi
komponen umum lalu membangun mekanisme komposisi View untuknya. Setiap langkah mengutip file nyata
sebagai bukti, bukan contoh karangan.

---

## Langkah 1 — Kumpulkan bukti dari BANYAK mockup, bukan satu

Jangan ekstrak komponen dari satu layar saja. Pola yang cuma muncul sekali bukan pola — itu
kebetulan. Method yang sudah terbukti di repo ini: kumpulkan semua mockup di `app/web/static/
ui-sample/*.html`, cari markup yang **berulang** secara visual, baru anggap itu kandidat komponen.

Bukti: `benchmarks/029-composed-view-component-inventory.md` (Study 38) mengklaster 14 mockup dan
menemukan 11 pola berulang — 9 di antaranya jadi komponen nyata, **1 ditolak** (`Choice Card` di
`document-submit.html`, cuma 1 kemunculan) dengan alasan eksplisit: "a single instance is not a
pattern." Itu bukan kegagalan — itu disiplin yang seharusnya. Jangan bangun komponen dari 1 bukti,
sekalipun mockup-nya terlihat meyakinkan.

**Konsolidasi, bukan cuma klaster.** Setelah menemukan pola berulang, uji dulu apakah SATU bentuk
komponen benar-benar bisa merender SEMUA sumbernya tanpa kehilangan sesuatu — jangan asumsikan.
`component-proof.html` (dibuat khusus untuk ini) menemukan hal nyata: `Sticky Action Bar`
awalnya diasumsikan "satu tombol yang disabled sampai syarat terpenuhi" (dari 2 sumber), tapi
sumber ketiga (`document-approval.html`) ternyata SELALU punya dua tombol aktif (Reject + Approve)
— bentuk komponennya harus diperbaiki jadi slot `actions []templ.Component` (0..N) SEBELUM kode Go
ditulis dari asumsi yang sempit. Diff markup itu murah — jalankan untuk tiap klaster sebelum jadi
komponen `templ` sungguhan, bukan cuma dua yang jadi bukti metode.

---

## Langkah 2 — Pisahkan dua kelas gap sebelum membangun apa pun

Setiap bagian mockup yang "belum bisa" masuk salah satu dari dua kelas, dan keduanya butuh
penanganan yang beda total:

| Kelas | Ciri | Contoh nyata | Yang harus dilakukan |
|---|---|---|---|
| **Presentation primitive** | Cuma soal bentuk visual — tidak butuh data/mekanisme baru | `Avatar`, `StickyActionBar`, `SectionWrapper` | Langsung jadi komponen `templ`, tidak perlu baris `capability-registry.md` baru |
| **Mekanisme metadata baru** | Butuh konsep baru di skema (Field/View/Event/Config) | `CAP-F24` (toggle User/Group per approver), `CAP-V10` Tier 2 (compose View) | **Wajib** lewat uji admisi A1–A5 (`capability-lifecycle.md` §2) dulu, baru dikerjakan |

Study 38 sendiri memisahkan ini dengan jelas: dari 11 klaster, cuma bagian yang "*is the rendering
mechanism CAP-V10 Tier 2 needs*" yang disentuh sebagai catatan kapabilitas; 9 sisanya murni tugas
implementasi `templ`. Jangan campur keduanya — komponen presentasi yang dipaksa lewat proses
admisi kapabilitas cuma buang waktu; mekanisme metadata baru yang dibangun TANPA uji admisi berisiko
jadi kapabilitas "hantu" tanpa bukti nyata (persis alasan `Choice Card` ditolak).

---

## Langkah 3 — Untuk presentation primitive: beri nama berdasar BENTUK, bukan use case

`RecordSummaryCard`, bukan `ApprovalListCard`. `StickyActionBar`, bukan `ApproveRejectBar`.
`SectionWrapper`, bukan `EmbeddedViewCard`. Nama berdasar bentuk visual (avatar + judul + subjudul
+ badge) membuat komponen itu otomatis terpakai ulang di konteks lain — persis yang membuat
`SectionWrapper` bisa dipakai untuk kartu Decision Progress DAN kartu Set Signature Position tanpa
tahu isi keduanya (lihat `internal/ui/components.templ`'s `SectionWrapper`, dipakai dari
`internal/ui/detail.templ`).

---

## Langkah 4 — Jangan taruh parameter yang bisa diturunkan dari parameter lain

Ini kesalahan konkret yang benar-benar terjadi dan harus diperbaiki di sesi yang sama:

```go
// SEBELUM — isPDF dihitung di handler, diteruskan lewat 3 lapis
func CoordPlacePreview(..., previewKey string, isPDF bool, ...) { ... }

// SESUDAH — previewKey saja; isPDF dihitung SENDIRI oleh layer render
func CoordPlacePreview(..., previewKey string, ...) {
    if isPDFPreview(previewKey) { ... }  // fungsi murni: filepath.Ext(previewKey) == ".pdf"
}
```

Kesalahan yang sama muncul dua kali dalam satu sesi (`isPDF` di `coordplace.templ`, lalu
`initials` di `MemberChip` — keduanya diturunkan langsung dari parameter lain yang sudah ada:
`previewKey`, `name`). **Aturan praktis:** sebelum menambah parameter baru ke sebuah komponen
render, tanya "bisakah ini dihitung dari parameter yang SUDAH ada di sini?" — kalau ya, hitung di
dalam, jangan minta pemanggil menghitung dan meneruskannya. Pemanggil yang menghitung ulang fakta
yang sama dengan cara berbeda adalah sumber bug diam-diam (dua perhitungan yang bisa saling tidak
sinkron), bukan cuma soal kerapian kode.

---

## Langkah 5 — Untuk komposisi View: dispatch berdasar TYPE, bukan flag per-kapabilitas

Ini pelajaran paling mahal di sesi ini — dilakukan salah **dua kali** sebelum benar. Kalau sedang
membangun "View A menampilkan View B secara inline," JANGAN membuat field config bernama sesuai
kapabilitas target (`parent_stepper_view`, `embedded_report_view`, dst — satu field baru per
kapabilitas). Itu tepat kebalikan dari kenapa View sengaja dipecah kecil-kecil: supaya metadata
yang mengomposisikannya, dengan runtime menentukan CARA render dari `type` milik View yang
direferensikan — bukan dari nama key metadata.

**Bentuk yang benar** (`internal/model/model.go`'s `ViewConfig.Children []ChildViewRef`):

```go
type ChildViewRef struct {
    View string `json:"view"`  // cuma id — TIDAK ADA opini soal jenisnya
}
```

Lalu SATU dispatcher generik (`internal/handler/embed.go`'s `renderChildView`):

```go
func (h *Handler) renderChildView(r *http.Request, hostRec *store.Record, childViewID string) *ui.EmbeddedSection {
    view, ok := h.interp.Get().GetView(childViewID)
    switch view.Type {                              // <-- dispatch dari TYPE View, bukan dari nama key
    case model.ViewTypeDecisionStepper:
        return h.renderDecisionStepperChild(r, hostRec, view)
    case model.ViewTypeCoordPlacement:
        return h.renderCoordPlacementChild(r, hostRec, view)
    default:
        return nil
    }
}
```

Ciri komponen ini bekerja dengan benar: menambah tipe kedua (`coord_placement`, ditambahkan sehari
setelah yang pertama) cuma butuh **satu baris `case`** baru plus satu entry di
`model.EmbeddableChildViewTypes` — tidak ada perubahan bentuk `Children` itu sendiri, tidak ada
field config baru. Kalau menambah kemampuan baru butuh field config baru per kapabilitas, itu
tanda arsitekturnya masih salah — kembali ke langkah ini.

---

## Langkah 6 — Komposisi HARUS dideklarasikan eksplisit; kode tidak boleh menebak

Kesalahan pertama (sebelum Langkah 5 di atas ditemukan): kode sempat menebak komposisi dengan
men-scan semua Machine mencari mana yang "cocok" secara struktur (`Machine.Config["steps_machine"]
== machineID`), lalu OTOMATIS menampilkan View lain begitu kecocokan ditemukan — tanpa satu pun
baris metadata yang benar-benar menyatakan "tampilkan ini." Ini salah secara arsitektural, bukan
cuma gaya: metadata tidak lagi bisa mematikan/memilih perilaku itu, karena perilakunya muncul dari
Go, bukan dari deklarasi.

**Uji cepat untuk komposisi apa pun:** kalau kamu menghapus SEMUA baris metadata yang relevan,
apakah tampilannya berubah? Kalau TIDAK berubah (masih tampil karena kode menebak dari struktur
data lain), itu salah. Kalau BERUBAH (hilang, karena tidak ada lagi yang mendeklarasikannya), itu
benar. Ini benar-benar diuji di sesi ini: config `children` di-reset ke `{}` pada binary yang
SAMA, kartu inline hilang total; `UPDATE` + `POST /admin/reload` (CAP-X04, tanpa restart, tanpa
deploy kode baru) mengembalikannya. Kalau perilaku HANYA bisa diubah lewat deploy kode baru, itu
bukan mekanisme metadata — itu kode yang menyamar sebagai metadata.

---

## Langkah 7 — Bedakan log: kondisi wajar vs anomali sungguhan

Setiap cabang "tidak ada yang ditampilkan" (`return nil`) di kode komposisi View HARUS dipikirkan:
apakah ini keadaan NORMAL (data belum lengkap, role tidak berhak baca) atau ANOMALI (metadata
menunjuk sesuatu yang seharusnya ada tapi tidak ada)? Yang normal — diam saja, log di sini cuma
jadi kebisingan di setiap page load biasa. Yang anomali — `slog.Warn` dengan detail spesifik (id
View, id record, field yang terlibat), bukan pesan generik.

Alasan ini penting, bukan sekadar rapi: sesi yang sama menemukan insiden nyata (`mch_ca_lifted`,
`capability-registry.md`'s baris `CAP-V20`) — kolom `process` yang korup diam-diam sampai server
gagal boot, TANPA satu pun log yang memperingatkan sebelumnya. Setiap jalur silent-`nil` yang
sebetulnya anomali adalah insiden itu, versi kecil, menunggu ditemukan.

---

## Langkah 8 — Jangan klaim "generik" sebelum ada pemakai kedua yang sungguhan

Menggeneralisasi sebuah mekanisme (Langkah 5) itu benar kalau alasannya prinsip arsitektur (bukan
spekulasi "mungkin nanti butuh") — tapi klaim itu tetap lemah sampai ada BUKTI, bukan cuma teori.
Ukurannya: cari atau bangun pemakai KEDUA yang sungguhan sesegera mungkin. Di sesi ini,
`coord_placement` ditambahkan sebagai tipe kedua yang bisa di-`children`-kan hari yang sama
mekanismenya dibuat — itu yang mengubah "generalisasi dibenarkan oleh prinsip" jadi "dibenarkan
oleh prinsip DAN praktik." Sebelum pemakai kedua itu ada, jujur saja tulis di dokumentasi bahwa
generalisasinya baru punya satu pemakai — jangan tutupi itu.

---

## Langkah 9 — Jalankan uji admisi sebelum membangun kapabilitas baru

Untuk kelas "mekanisme metadata baru" (Langkah 2), jalankan `capability-lifecycle.md` §2 A1–A5
secara jujur sebelum menulis kode:

- **A1** — bukti ganda (case + benchmark/pola industri), bukan cuma satu mockup.
- **A2** — universalitas (bukan spesifik ke satu vertikal bisnis).
- **A3** — tanggung jawab tunggal (Field/Event/Constraint/View — bukan campuran).
- **A4** — tidak bisa disusun dari kapabilitas yang sudah ada (`Choice Card` gagal di sini kalau
  cuma soal styling `value_list`, bukan mekanisme baru).
- **A5** — bisa dijelaskan dalam bahasa bisnis biasa, bukan istilah implementasi.

Kalau gagal salah satu — catat sebagai **"not admitted"** dengan alasannya, JANGAN dibangun, dan
JANGAN didiamkan begitu saja (prinsip "silence is not a decision" — `capability-registry.md`'s
§Tracked but Not Yet Studied adalah contoh cara mencatat gap yang belum lolos uji tanpa
melupakannya).

---

## Checklist ringkas

Sebelum menganggap satu bagian mockup "selesai jadi komponen metadata":

- [ ] Polanya muncul di ≥2 mockup atau kasus nyata, bukan cuma satu (Langkah 1)
- [ ] Sudah dipisahkan: presentation primitive vs mekanisme metadata baru (Langkah 2)
- [ ] Kalau presentation primitive: nama berdasar bentuk, sudah diuji render dari ≥2 sumber nyata
      (Langkah 1, 3)
- [ ] Tidak ada parameter yang bisa diturunkan dari parameter lain yang sudah ada (Langkah 4)
- [ ] Kalau ini komposisi View: dispatch dari `type` View yang direferensikan, bukan dari nama key
      metadata (Langkah 5)
- [ ] Menghapus metadata-nya benar-benar mematikan tampilannya — sudah diuji langsung, bukan
      diasumsikan (Langkah 6)
- [ ] Setiap jalur "tidak tampil" sudah dibedakan: wajar (diam) vs anomali (log spesifik) (Langkah 7)
- [ ] Kalau baru satu pemakai: sudah ditulis jujur sebagai "generalized by principle, not yet by
      practice," bukan diklaim generik penuh (Langkah 8)
- [ ] Kalau ini kapabilitas baru: sudah lolos A1–A5, tercatat di `capability-registry.md` — atau
      tercatat honestly sebagai "not admitted" kalau gagal (Langkah 9)

## Referensi implementasi lengkap (studi kasus sesi ini)

`internal/model/model.go` (`ViewConfig.Children`, `ChildViewRef`, `EmbeddableChildViewTypes`),
`internal/handler/embed.go` (dispatcher generik), `internal/handler/decisionstepper.go` +
`internal/handler/coordplace.go` (dua implementasi per-tipe + logging yang dibedakan),
`internal/metadata/validate.go` (validasi load-time), `internal/ui/components.templ` (9 primitif
presentasi Study 38), `app/docs/ui-component-library.md` (katalog lengkap + riwayat koreksi
berstempel tanggal), `capability-registry.md`'s baris `CAP-V20` (riwayat dua koreksi arsitektur
dalam satu sesi, dicatat apa adanya, tidak ditutupi).
