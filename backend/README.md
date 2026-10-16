# SPKD MedPay API

Backend **SPKD MedPay** (Go + Gin + GORM + PostgreSQL). Melayani frontend Vue (`../frontend`) lewat REST API di
`/api`, dengan kontrak yang sama persis dengan mock API frontend (`frontend/src/mock/server.js`).

Fungsinya: klaim selisih biaya (tagihan RS − tarif JKN), resolusi tarif INA-CBG, scrubber 14 aturan, batch pengajuan,
pembayaran & rekonsiliasi mutasi bank, piutang (aging/DSO), simulasi KAPJ, antrean kerja, audit trail, dan
monitoring Engine API. Multi-RS: data tiap RS terisolasi, dan setiap perubahan tercatat di `audit_logs`.

## Struktur Folder

```
backend/
├── cmd/server/main.go        # titik masuk: migrasi, seed demo, server HTTP
├── internal/
│   ├── api/                  # handler HTTP & router
│   │   ├── router.go         # daftar endpoint + middleware keamanan
│   │   ├── auth_handler.go   # login, MFA, logout + pembatas percobaan login
│   │   ├── fe_*.go           # endpoint yang dipakai frontend Vue
│   │   ├── dto.go            # pemetaan model → JSON frontend
│   │   └── dataset.go        # pemuatan data per RS untuk perhitungan agregat
│   ├── auth/                 # JWT + tabel sessions, sandi Argon2id, MFA kode demo, RBAC, isolasi RS
│   ├── audit/                # pencatatan audit trail
│   ├── db/                   # koneksi database + migrasi (AutoMigrate)
│   ├── engine/               # logika bisnis murni + unit test (tarif, scrubber, KAPJ, aging, rekonsiliasi)
│   ├── model/                # struct GORM per tabel
│   ├── seed/                 # seeder: data master (selalu) + data demo (opsional)
│   └── appenv/               # helper environment
└── go.mod / go.sum
```

## Menjalankan

1. Buat database PostgreSQL, salin `.env.example` → `.env`, lalu isi `DB_*` dan `JWT_SECRET`.
2. Jalankan:
   ```bash
   go run ./cmd/server          # migrasi + seed demo (bila SEED_DEMO=true & DB kosong) + server :8080
   go run ./cmd/server -fresh   # HAPUS semua tabel, migrasi & seed ulang (dev saja)
   ```
3. Cek: `curl http://localhost:8080/api/health` → `{"database":"up","status":"up"}`

Frontend: set `VITE_USE_MOCK=false` di `frontend/.env.local`, lalu `npm run dev` (proxy `/api` → `:8080`).

Tabel dibuat otomatis saat start. Database yang dibuat sebelum primary key berubah ke UUID (ID angka) tidak bisa
dikonversi otomatis: server menolak start dan meminta `go run ./cmd/server -fresh` (menghapus semua data).

## Konfigurasi `.env`

| Variabel | Keterangan |
|---|---|
| `APP_ENV` | `development` / `production` |
| `APP_PORT` | Port server, default `8080` |
| `SEED_DEMO` | `true` = isi data demo bila DB kosong. Ditolak di produksi. |
| `CORS_ORIGIN` | Origin frontend, pisahkan dengan koma. Default `http://localhost:5173` |
| `TRUSTED_PROXIES` | IP reverse proxy. Kosong = tidak percaya proxy mana pun |
| `JWT_SECRET` | Wajib di produksi, minimal 32 karakter |
| `INITIAL_ADMIN_EMAIL` `INITIAL_ADMIN_PASSWORD` `INITIAL_ADMIN_NAME` | Membuat Super Admin pertama bila tabel users kosong (instalasi produksi tanpa data demo); wajib ganti sandi saat masuk |
| `MFA_DEMO_CODE` | Kode MFA langkah kedua di lingkungan non-produksi, default `246810` |
| `SESSION_IDLE_MINUTES` | Sesi berakhir setelah tidak ada aktivitas, default `30` (F-AUT-10). `0` = nonaktif |
| `AUTO_TASKS_INTERVAL_MIN` | Interval pembuatan tugas otomatis antrean kerja, default `60` (F-WQ-04). `0` = nonaktif |
| `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` | Koneksi PostgreSQL |
| `DB_SSLMODE` | `disable` lokal, `require` produksi |
| `DB_TIMEZONE` | Zona waktu "hari ini", deadline & aging. Default `Asia/Jakarta` |

Saat `APP_ENV=production`, server menolak start bila `JWT_SECRET` < 32 karakter, `DB_PASSWORD` < 16 karakter,
`SEED_DEMO=true`, atau `DB_SSLMODE=disable` ke database yang bukan lokal.

## Data & Seeder

Semua data berasal dari PostgreSQL. Backend tidak membaca file JSON/mock apa pun saat berjalan. Seeder Go di
`internal/seed/` terbagi dua:

**1. Data master — selalu diisi setiap server start (juga di produksi).** Hanya baris yang belum ada yang
ditambahkan; data yang sudah diubah admin tidak ditimpa.

| File | Isi |
|---|---|
| `master_data.go` | 13 izin, 10 peran + matriks izin (SRS 2.3), 7 kategori resume, 8 kode penolakan, 14 aturan scrubber R01–R14, 106 kode referensi (24 kategori), 4 pengaturan aplikasi |
| `master.go` | `Master()` (pengisian idempoten) dan `BootstrapAdmin()` (Super Admin awal dari `INITIAL_ADMIN_*`) |

**2. Data demo — hanya bila `SEED_DEMO=true` dan database kosong, atau `go run ./cmd/server -fresh`.**

| File | Isi |
|---|---|
| `seed_org.go` | 4 rumah sakit, 10 pengguna, 14 penjamin, 7 regulasi |
| `seed_claims.go` | 3 batch, 34 klaim (diagnosa, prosedur, tarif, riwayat status), 3 override scrubber |
| `seed_resume.go` | 122 item resume medis |
| `seed_finance.go` | 6 pembayaran, 2 impor mutasi, 9 baris mutasi |
| `seed_ops.go` | 6 tugas antrean kerja, 4 notifikasi, 25 audit log |
| `seed_platform.go` | 2 partner Engine API (+ user key per RS), 14 klaim masuk via Engine API |
| `seed_extra.go` | Tambahan: 13 klaim bersih lintas status (Draft → Lunas, ditolak, disanggah, hapus buku), 4 batch per-penjamin (BT-0004..0007), 3 pembayaran, 1 impor mutasi (3 baris), 6 tugas, 3 notifikasi. Klaim dibangun dari tabel `extraSpecs` |
| `seed_test.go` | Tes integritas data demo tanpa database (referensi, ID ganda, klaim tambahan bebas blocker) — jalankan `go test ./internal/seed/` setelah mengubah data demo |
| `steps.go` | Langkah pengisian & relasi antar tabel |

Setelah seed, semua tabel terisi kecuali `sessions` (terbentuk saat login). Tanggal transaksi demo ditulis relatif
terhadap `seedDate` dan digeser ke hari ini saat seeding, agar deadline, aging, dan KPI tetap bermakna.

### Data referensi untuk frontend

`GET /reference` (juga ikut di `GET /bootstrap` sebagai `reference`) mengembalikan semua yang dibutuhkan
tampilan, sehingga frontend tidak perlu menulis label/pilihan sendiri:

| Kunci | Isi |
|---|---|
| `codes.<kategori>` | Daftar pilihan berurutan: `code`, `label`, `tone` (warna badge), `description`, `active` |
| `labels.<kategori>` | Peta `code → label` untuk tampilan cepat |
| `settings` | Teks aplikasi: `app_name`, `app_tagline`, `support_contact`, `kapj_disclaimer` |
| `roles`, `permissions` | Peran beserta izinnya, dan label izin |
| `resumeCategories`, `rejectionCodes`, `scrubRules` | Master untuk form & validasi |
| `claimTransitions`, `transitionPermissions` | Diagram status B.3 dan izin per status tujuan |
| `editableStatuses`, `payableStatuses` | Status yang boleh diedit / menerima pembayaran |
| `businessRules` | Angka aturan SRS: deadline 6 bulan & zona, skor kesiapan, bobot pencocokan mutasi, bucket aging, KAPJ default, ambang warna performa penjamin, panjang minimal sandi, timeout sesi |

Kategori kode: `claim_status`, `claim_action`, `tariff_component`, `special_cmg`, `care_type`, `care_class`,
`participant_type`, `sex`, `insurance_type`, `coverage_type`, `hospital_class`, `insurer_type`,
`work_item_status`, `work_item_priority`, `work_item_source`, `deadline_zone`, `match_result`,
`payment_allocation`, `audit_scope`, `intake_status`, `regulation_status`, `notification_channel`,
`notification_status`, `scrub_severity`.

Selain itu, setiap klaim membawa `hospitalTotal`, `jknTotal`, `gap`, `deadline`, `deadlineDaysLeft`, dan
`deadlineZone`. Detail klaim membawa `transitions` (aksi status beserta label, warna, teks konfirmasi, izin,
dan apakah pengguna saat ini boleh menjalankannya), `editable`, dan `deletable`.

Label, warna, urutan, dan teks dapat diubah Super Admin lewat `PUT /reference/:category/:code` dan
`PUT /app-settings/:key` tanpa mengubah kode (N-MNT-01).

Yang tetap ditulis di kode karena merupakan logika, bukan data: aturan bisnis SRS (transisi status, skor,
pencocokan, zona deadline, bucket aging) dan grouper INA-CBG simulasi (`engine.DemoGrouper`) sampai integrasi
e-Klaim tersedia. Angka-angkanya tetap diekspos lewat `businessRules`.

## Akun Demo

Password semua akun `medpay2026`. Akun ber-MFA memakai kode demo `246810` (`MFA_DEMO_CODE`, hanya di luar
produksi): `superadmin@spkd.co.id`, `admin@nusantara.id`, `manajemen@nusantara.id`, `finance@nusantara.id`.
Tanpa MFA: `billing@`, `coder@`, `reviewer@`, `auditor@`, `it@`, `viewer@nusantara.id`.

Kredensial Engine API demo (dibentuk dari consumer ID partner, lihat `internal/seed/steps.go`):
consumer secret `demo-cs-<consumerId>`, user key `demo-uk-<consumerId>-<kodeFaskes>`. IP pemanggil harus ada di
whitelist partner.

## Ringkasan API (`/api`)

Respons sukses berupa JSON data langsung; gagal berupa `{"error": "pesan"}`. Token dikirim sebagai
`Authorization: Bearer <token>`, atau otomatis lewat cookie httpOnly `medpay_session` yang diset saat login.

| Method | Endpoint | Izin |
|---|---|---|
| POST | `/auth/login`, `/auth/verify`, `/auth/logout` | publik |
| GET | `/me`, `/bootstrap`, `/reference` | login |
| PUT · POST | `/notifications/:id/read` · `/notifications/read-all` | login (notifikasi sendiri) |
| PUT | `/reference/:category/:code`, `/app-settings/:key` | CONFIGURE + Super Admin |
| POST | `/auth/change-password` | login (akun sendiri) |
| GET | `/dashboard`, `/analytics`, `/aging` · POST `/kapj/simulate` | VIEW |
| GET/POST | `/claims` · GET/PUT/DELETE `/claims/:id` | VIEW / CREATE / EDIT / DELETE |
| POST | `/claims/:id/resolve` · `/grouper/preview` | RESOLVE |
| POST | `/claims/:id/transition` | per transisi (lihat di bawah) |
| POST | `/claims/:id/scrub` · GET `/scrubber/top-causes` | VIEW |
| PUT | `/scrub-rules/:id` · POST `/scrub-rules/override` | CONFIGURE · OVERRIDE |
| POST/PUT/DELETE | `/resume/bulk`, `/resume/:id` | CREATE / EDIT / DELETE |
| GET/POST/DELETE | `/batches`, `/batches/:id`, `/batches/:id/claims[/:claimId]` | VIEW / CREATE / EDIT / DELETE |
| POST | `/batches/:id/submit` (ajukan seluruh batch) | APPROVE |
| GET/POST/DELETE | `/payments`, `/payments/:id` | VIEW / CREATE / EDIT |
| GET/POST | `/bank-imports`, `/bank-lines`, `/bank-lines/:id/candidates` | VIEW / CREATE |
| POST | `/bank-lines/:id/confirm`, `/bank-lines/:id/unconfirm` | APPROVE |
| GET/POST/PUT | `/work-items`, `/work-items/:id`, `/work-items/:id/comments` | VIEW / CREATE / EDIT |
| GET | `/audit`, `/engine-api/summary` | VIEW |
| POST/PUT | `/hospitals`, `/users`, `/users/:id/reset-password`, `/insurers`, `/insurers/:id` | CONFIGURE |
| PUT/POST | `/partners/:id`, `/partners/:id/keys`, `/partners/:id/rotate-secret` | CONFIGURE + Super Admin |
| GET | `/export/claims.csv`, `/export/aging.csv`, `/export/audit.csv` | EXPORT |
| POST | `/engine/claims` (SIMRS partner) | header `X-Consumer-Id`, `X-Consumer-Secret`, `X-User-Key` |

**ID:** semua `id` (di respons, parameter URL, dan body seperti `claimId`, `claimIds`, `batchId`,
`importId`, `hospitalId`, `assignee`) berupa UUID v7. Nomor dokumen yang mudah dibaca dikirim sebagai field
terpisah dan dipakai untuk tampilan & pencarian: `claimNo` (CLM-0001), `batchNo`, `paymentNo`, `itemNo`,
`importNo`, `lineNo`, `intakeNo`. Audit trail mencatat nomor dokumen ini di `entityId`.

Staf SPKD (akun tanpa RS) melihat semua RS dan bisa memfilter dengan `?hospitalId=`. Akun RS terkunci ke RS-nya.
Setiap penolakan izin tercatat di audit sebagai `ACCESS_DENIED`.

### Status klaim (SRS Gambar B.3)

| Dari | Ke | Izin |
|---|---|---|
| Draft ⇄ Siap Diajukan | — | EDIT (wajib tanpa blocker & pernyataan dicentang) |
| Siap Diajukan / Disanggah | Diajukan | APPROVE, pengaju ≠ pembuat klaim, tanpa blocker |
| Diajukan | Disetujui / Disetujui Sebagian / Ditolak | VERIFY (Ditolak wajib kode alasan) |
| Disetujui Sebagian | Ditolak | VERIFY |
| Ditolak | Disanggah | REOPEN |
| Draft, Siap Diajukan, Ditolak, Disanggah, Sebagian Dibayar, Lunas | Dihapus Buku (final) | APPROVE |

Sebagian Dibayar dan Lunas tidak bisa dipilih manual: keduanya dihitung otomatis dari pembayaran (BR-05).

## Kesesuaian dengan SRS v1.0

Celah implementasi di SRS bab 6.3 yang sudah ditangani backend ini:

| ID | Penanganan |
|---|---|
| F-AUT-05 | `GET /me` memuat ulang user & izin dari database |
| F-AUT-10 | Sesi berakhir setelah 30 menit tanpa aktivitas (`SESSION_IDLE_MINUTES`) |
| F-CLM-09 | Hapus klaim hanya Draft, item resume ikut terhapus |
| F-PAY-05 | Kandidat & auto-match rekonsiliasi hanya klaim RS pengimpor |
| F-STS-04 | Pengaju ≠ pembuat klaim (maker-checker) |
| F-TRF-04 | Grouper dijalankan di server untuk form (`/grouper/preview`) dan saat klaim disimpan |
| F-BAT-04 | `POST /batches/:id/submit` mengajukan seluruh klaim siap di batch |
| F-WQ-04 | Tugas otomatis: klaim ditolak, mutasi perlu review (langsung); blocker, deadline kritis, piutang > 90 hari (berkala) |
| F-ADM-05/06 | Engine API partner: consumer secret, user key per RS, IP whitelist, rate limit, grouper + KAPJ |
| F-ADM-02 | Status aktif pengguna, reset sandi, pemberian peran dibatasi izin pengelola |
| F-AUD-03 | Login/MFA gagal, akun terkunci, resolusi gagal, dan akses ditolak tercatat permanen |
| N-PRF-02 | PostgreSQL + transaksi per perubahan |
| N-SEC-01 | Argon2id (hash bcrypt lama otomatis di-upgrade saat login) |
| N-SEC-02 | Cookie httpOnly + SameSite=Strict (frontend masih membaca token dari body, lihat catatan) |
| N-SEC-04 | Sesi disimpan di tabel `sessions`, tetap berlaku saat server restart |
| N-SEC-05 | Akun terkunci 15 menit setelah 5 kali gagal berturut-turut |

Belum ditangani: F-AUT-02 (MFA masih kode demo; TOTP sengaja tidak dipakai, sehingga login MFA
ditolak di produksi), F-CLM-13 & F-BAT-03 (PDF dibuat server), integrasi e-Klaim asli (grouper masih demo),
N-SEC-03 (HTTPS diatur di reverse proxy), N-PRF-01 (uji beban), N-MNT-02/03 (kajian hukum & masa simpan).

## Testing

```bash
go vet ./...
go test ./...
```

Unit test ada di `internal/engine` (rantai tarif, 14 aturan scrubber, deadline, KAPJ, aging/DSO, rekonsiliasi,
grouper, transisi status) dan `internal/auth` (MFA demo, Argon2id, kebijakan sandi, pemberian peran, isolasi RS).

Uji API end-to-end: `postman/collections/SPKD MedPay API.postman_collection.json` (121 request, 205 assertion),
environment `postman/environments/MedPay Lokal.postman_environment.json`. Jalankan dengan Postman Runner atau:

```bash
npx newman run "postman/collections/SPKD MedPay API.postman_collection.json" \
  -e "postman/environments/MedPay Lokal.postman_environment.json"
```

Collection memakai data demo (`SEED_DEMO=true`) dan aman dijalankan berulang.
