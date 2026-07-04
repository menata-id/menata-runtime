# Menata Language — Grammar Reference

Menata Language adalah cara menulis Business Knowledge dalam bentuk teks biasa.
File berekstensi `.menata`.

File ini adalah **sumber kebenaran** (source of truth). Runtime Metadata (YAML/DB) adalah
realisasinya — diturunkan dari file ini, bukan sebaliknya.

---

## Struktur Dokumen

```
<nama mesin>

Fields
<daftar field>

Events
<daftar event>

Constraints
<daftar constraint>

Permissions
<daftar permission per role>

Views
<daftar view>
```

Urutan bagian tidak diwajibkan, tapi konvensi di atas direkomendasikan.

---

## Nama Mesin

Baris pertama dokumen adalah nama mesin. Ditulis dalam bahasa bisnis, bukan teknis.

```
Design Request
```

```
Leave Request
```

---

## Fields

Mendaftarkan data apa yang dimiliki mesin ini.

```
Fields

- <Nama Field> : <Tipe>
```

### Tipe Field

| Sintaks | Tipe | Keterangan |
|---------|------|-----------|
| `Text` | text | Teks pendek satu baris |
| `Rich Text` | rich_text | Teks panjang / paragraf |
| `Number` | number | Angka |
| `Money` | money | Nilai uang |
| `Boolean` | boolean | Ya / Tidak |
| `Date` | date | Tanggal |
| `Time` | time | Waktu |
| `Date Time` | date_time | Tanggal dan waktu |
| `Duration` | duration | Rentang waktu |
| `User` | user | Referensi ke pengguna |
| `File` | file | Lampiran file |
| `<A> \| <B> \| <C>` | value_list | Pilihan dari daftar tetap |
| `Reference: <Nama Mesin>` | reference | Referensi ke mesin lain |

### Contoh

```
Fields

- Requester : User
- Design Type : Poster | Thumbnail | Banner 2:1
- Due Date : Date
- Title : Text
- Description : Rich Text
- Attachment : File
```

**Catatan:** Field `Status` tidak perlu ditulis di bagian Fields — runtime
menambahkannya secara otomatis berdasarkan nilai-nilai yang muncul di Events.
Jika perlu kontrol eksplisit atas urutan nilai status, tulis secara manual.

---

## Events

Mendaftarkan kejadian bisnis yang dapat terjadi pada mesin ini.

```
Events

When <Nama Event>

    <Aksi>
    <Aksi>
    ...
```

### Aksi

| Sintaks | Keterangan |
|---------|-----------|
| `Status <Nilai>` | Ubah status menjadi nilai tersebut |
| `Notify <Role>` | Kirim notifikasi ke role tersebut |
| `Record <Nama>` | Catat ke log / register |

### Contoh

```
Events

When Submit

    Status Submitted

    Notify Design Team

When Accept

    Status Accepted

When Complete

    Status Completed

    Notify Requester
```

---

## Constraints

Mendaftarkan aturan bisnis yang harus selalu dipenuhi sebelum data disimpan.

Ditulis sebagai kalimat bahasa alami yang berakhir dengan tanda titik.

```
Constraints

- <Aturan bisnis.>

- <Aturan bisnis.>

    if <Field> = <Nilai>
```

Baris `if` menjadikan constraint bersifat kondisional — hanya berlaku
ketika kondisi tersebut terpenuhi.

### Contoh

```
Constraints

- Title is required.
- Description is required.
- Due Date must be after today.
- Attachment is required.

    if Design Type = Banner 2:1
```

### Operator yang didukung runtime saat ini

| Kalimat | Operator | Keterangan |
|---------|----------|-----------|
| `<Field> is required.` | `required` | Field tidak boleh kosong |
| `<Field> must be after today.` | `after: today` | Tanggal harus di masa depan |
| `<Field> = <Nilai>` *(di kondisi `if`)* | `equals` | Perbandingan nilai |
| `<Field> ≠ <Nilai>` *(di kondisi `if`)* | `not_equals` | Nilai bukan X |

Operator baru ditambahkan di runtime tanpa perubahan sintaks Language.

---

## Permissions

Mendaftarkan siapa (role) yang boleh memicu event apa.

```
Permissions

<Role>

- <Event>
- <Event>

<Role>

- <Event>
```

### Contoh

```
Permissions

Requester

- Submit

Designer

- Accept
- Reject
- Start
- Complete
```

Role ditulis bebas — tidak ada daftar tetap. Runtime menggunakan nilai role
yang sama persis dengan yang tercatat di metadata permissions.

---

## Views

Mendaftarkan cara mesin ini ditampilkan.

```
Views

- <Nama> : <Tipe>
```

### Tipe View

| Tipe | Keterangan |
|------|-----------|
| `Form` | Formulir input untuk membuat atau mengubah record |
| `List` | Tabel atau kartu untuk menampilkan banyak record |
| `Detail` | Tampilan lengkap satu record |
| `Dashboard` | Ringkasan dan metrik |
| `Calendar` | Tampilan berbasis tanggal |
| `Timeline` | Tampilan kronologis |

### Contoh

```
Views

- Request Form : Form
- My Requests : List
- Request Detail : Detail
- Design Queue : List
```

Konfigurasi kolom, field yang tampil di form, dan urutan default ditentukan
di Runtime Metadata (YAML) — bukan di file `.menata`. File `.menata` hanya
menyatakan keberadaan view dan tipenya.

---

## Konvensi Penulisan

- Nama field, event, role, dan view ditulis dalam **Title Case**.
- Kalimat constraint diakhiri **tanda titik**.
- Indentasi aksi di dalam event menggunakan **4 spasi**.
- Tidak ada batas panjang baris, tapi jaga agar kalimat tetap ringkas.
- File disimpan sebagai UTF-8, line ending LF.

---

## Hubungan dengan Runtime Metadata

File `.menata` ditulis oleh domain expert.
Runtime Metadata (YAML / DB) diturunkan dari file ini oleh pengembang.

```
leave-request.menata   ←  domain expert menulis ini
leave-request.yaml     ←  pengembang menerjemahkan ke sini
seeds/002_*.sql        ←  pengembang memasukkan ke DB
```

Lihat `docs/examples/` untuk contoh lengkap keduanya berdampingan.
