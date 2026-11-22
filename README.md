# BIOTA Marketplace Sales Insight

Aplikasi untuk mengunggah file CSV pesanan gabungan dari Shopee, TikTok Shop, dan Tokopedia, meninjau masalah data, lalu melihat **pendapatan bersih** per channel dan per produk, Top 10 produk, serta tren harian. Semua tampilan bisa difilter berdasarkan tanggal dan channel.

🔗 **Aplikasi live: https://biota.azis.works**

Dokumen kebutuhan lengkap: [`docs/2026-11-23-marketplace-sales-insight/prd.md`](docs/2026-11-23-marketplace-sales-insight/prd.md).

---

## Menjalankan Aplikasi Secara Lokal

### Opsi 1: Go (untuk pengembangan)

Prasyarat:
- Go ≥ 1.26
- `templ`: `go install github.com/a-h/templ/cmd/templ@v0.3.977`
- Tailwind CSS standalone CLI: `make tailwind-install` (diunduh ke `./bin`, cukup sekali)

```sh
make tailwind-install   # sekali saja
make run                # templ generate + build CSS, lalu jalankan dengan DATA_DIR=./.data
```

Buka http://localhost:8080, lalu unggah file CSV lewat menu **Unggah Data**.

### Opsi 2: Docker

```sh
docker compose up -d --build
# buka http://localhost:8080
```

Atau build image-nya saja:

```sh
make docker-build
docker run -d -p 8080:8080 -v biota-data:/data biota-insight
```

### Variabel Lingkungan

| Variabel | Default | Keterangan |
|---|---|---|
| `PORT` | `8080` | Port HTTP |
| `DATA_DIR` | `/data` | Folder untuk `dataset.json` dan `config.json` |
| `MAX_UPLOAD_MB` | `10` | Batas ukuran file unggahan (MB) |

### Perintah `make`

| Perintah | Fungsi |
|---|---|
| `make generate` | `templ generate` dan build `static/css/app.css` |
| `make test` | `go test -count=1 -failfast ./...` (jalankan `make generate` dulu di checkout baru) |
| `make run-lint` | `golangci-lint run ./...` |
| `make mocks` | Generate ulang `services/sales/mocks` dengan mockery |
| `make docker-build` | Build image `biota-insight` |

---

## Hasil Analisis: Pesanan Oktober 2026

Angka di bawah berasal dari file ekspor asli Oktober 2026 (495 baris). File itu diproses dengan pilihan default di tahap review: 22 baris duplikat persis dibuang, dan status mengikuti pemetaan default (lihat [Keputusan Terkait Data](#keputusan-terkait-data)).

### Periode 1–31 Oktober 2026 (semua channel)

| Metrik | Nilai |
|---|---|
| **Total Pendapatan Bersih** | **Rp298.937.000** |
| Pesanan | 366 |
| Unit Terjual | 473 |
| Rata-rata Nilai Pesanan (AOV) | Rp816.768 |

**Pendapatan Bersih per Channel**

| Channel | Pesanan | Unit | Pendapatan Bersih | % |
|---|---:|---:|---:|---:|
| Shopee | 223 | 295 | Rp180.315.000 | 60,3% |
| TikTok Shop | 80 | 99 | Rp70.461.000 | 23,6% |
| Tokopedia | 63 | 79 | Rp48.161.000 | 16,1% |
| **Total** | **366** | **473** | **Rp298.937.000** | **100%** |

**Pendapatan Bersih per Produk** (10 teratas = Top 10 Produk)

| Rank | SKU | Nama Produk | Unit | Pendapatan Bersih | % |
|---:|---|---|---:|---:|---:|
| 1 | BIO-CAM-BF | BIOTA Smart Camera Battery Fixed - CCTV Wifi Tanpa Kabel | 55 | Rp60.445.000 | 20,2% |
| 2 | BIO-CAM-E4G | BIOTA Smart Camera Edge 4G PTZ + Solar Panel | 18 | Rp39.582.000 | 13,2% |
| 3 | BIO-PET-FD | BIOTA Smart Pet Feeder - Dispenser Makanan Otomatis | 41 | Rp36.859.000 | 12,3% |
| 4 | BIO-CAM-BBY | BIOTA Smart Baby Camera + Monitor Tablet | 28 | Rp36.372.000 | 12,2% |
| 5 | BIO-CAM-OPZ | BIOTA Smart Camera Outdoor PTZ - CCTV Outdoor Waterproof | 40 | Rp27.960.000 | 9,4% |
| 6 | BIO-CAM-DB | BIOTA Smart Doorbell Wireless Camera | 32 | Rp25.568.000 | 8,6% |
| 7 | BIO-CAM-IPZ | BIOTA Smart Camera Indoor PTZ - Kamera CCTV 360 | 41 | Rp18.409.000 | 6,2% |
| 8 | BIO-CAM-IPL | BIOTA Smart Camera Indoor PTZ Lite | 40 | Rp13.960.000 | 4,7% |
| 9 | BIO-SD-V128 | MicroSD V-GeN Turbo Series 128GB | 69 | Rp10.971.000 | 3,7% |
| 10 | BIO-PET-WF | BIOTA Smart Pet Water Fountain | 27 | Rp10.773.000 | 3,6% |
| 11 | BIO-SD-S64 | BIOTA MicroSD SONIC Series 64GB | 76 | Rp9.044.000 | 3,0% |
| 12 | BIO-LGT-HEX | BIOTA Lumi Hexagon Light Panel | 6 | Rp8.994.000 | 3,0% |

### Periode 1–15 Oktober 2026 (semua channel)

| Metrik | Nilai |
|---|---|
| **Total Pendapatan Bersih** | **Rp146.226.000** |
| Pesanan | 187 |
| Unit Terjual | 234 |
| Rata-rata Nilai Pesanan (AOV) | Rp781.957 |

| Channel | Pesanan | Unit | Pendapatan Bersih | % |
|---|---:|---:|---:|---:|
| Shopee | 113 | 140 | Rp83.050.000 | 56,8% |
| TikTok Shop | 37 | 48 | Rp36.812.000 | 25,2% |
| Tokopedia | 37 | 46 | Rp26.364.000 | 18,0% |
| **Total** | **187** | **234** | **Rp146.226.000** | **100%** |

> Catatan: file ekspor asli **tidak** disimpan di repo. Pengujian otomatis memakai data sintetis `test_data/sample_orders_oct2026.csv`, yang strukturnya sama dengan data asli tetapi isinya acak, sehingga angkanya berbeda dengan tabel di atas.

---

## Keputusan Terkait Data

Keputusan berikut diambil saat diskusi kebutuhan, berdasarkan kondisi data di file ekspor Oktober 2026.

### Definisi pendapatan
- **Pendapatan bersih = jumlah `total_price`** dari baris dengan status *sedang dikirim* atau *selesai*. Tidak ada pengurangan biaya marketplace, voucher, atau ongkir, karena kolomnya tidak ada di CSV.
- **Status yang dihitung**: Selesai, Completed, Pesanan Selesai, Dikirim, In transit, Sedang Dikirim.
- **Status yang tidak dihitung** (batal/retur): Batal, Cancelled, Dibatalkan, Pengembalian Dana, Refunded, Dikomplain - Retur.
- **Status baru** yang belum dikenal muncul di tahap review, dan pengguna menentukan dihitung atau tidak. Pilihan disimpan dan bisa diubah di **Pengaturan → Status**. Perubahan langsung berlaku untuk data saat ini.
- **Pesanan** = jumlah `order_id` unik (satu pesanan bisa berisi beberapa produk). **AOV** = pendapatan bersih ÷ pesanan.
- Hanya KPI bersih yang ditampilkan (tanpa pendapatan kotor atau rasio batal/retur).

### Normalisasi data
- **Channel** ditulis dalam 10 variasi (mis. `shopee`, `SHOPEE`, `Tokped`, `Tiktok`) dan disatukan menjadi Shopee, TikTok Shop, dan Tokopedia.
- **Format tanggal berbeda per channel**: Shopee `2026-10-21 08:24`, TikTok `14/10/2026 23:26:00` (hari/bulan), Tokopedia `Oct 01, 2026` (tanpa jam). Tanggal disimpan apa adanya tanpa konversi zona waktu.
- **Format harga** ada sekitar 6 variasi (`Rp1.099.000`, `Rp 1.099.000`, `1,099,000`, `IDR 1,099,000`, `1099000`), semuanya diubah ke rupiah bulat.
- **SKU kosong** (76 baris) diisi dari `product_name`, memakai pasangan nama→SKU di file yang sama atau pemetaan tersimpan. Jika tetap tidak dikenali, pengguna memilih SKU di tahap review dan pilihannya diingat untuk unggahan berikutnya.
- **Produk dikelompokkan per SKU**, karena satu SKU punya 3 nama listing berbeda di tiap marketplace. Nama tampilan default adalah nama listing yang paling sering muncul, dan bisa diubah di **Pengaturan → Produk**.

### Kualitas data dan tahap review
- **Baris duplikat persis** (22 baris di data Oktober) ditampilkan per grup di tahap review, dan **default-nya dibuang**. Pengguna bisa memilih menyimpannya.
- **Baris yang gagal dibaca** (format tanggal/harga/channel tidak dikenal, qty tidak valid, order_id kosong) ditampilkan dengan nomor baris dan alasannya, lalu **dilewati** saat konfirmasi.
- **Peringatan (tidak memblokir impor)**: jika `qty × unit_price ≠ total_price`, atau prefix `order_id` (SHP/TTS/TKP) tidak sesuai channel. Baris tetap diimpor apa adanya.

### Penyimpanan dan tampilan
- **Tanpa database.** Setiap unggahan yang dikonfirmasi **menggantikan seluruh data sebelumnya**, jadi hanya CSV terakhir yang dianalisis. Data terakhir dan pengaturan disimpan sebagai file JSON di volume, sehingga tetap ada setelah restart.
- **Filter**: rentang tanggal per hari (inklusif, default seluruh periode data, tanpa preset) dan pilihan channel.
- **Tampilan**: kartu KPI, tabel pendapatan per channel dan per produk, grafik Top 10 produk, dan grafik tren harian (Chart.js).
- Satu pengguna, tanpa login. Antarmuka dalam Bahasa Indonesia, dan format uang `Rp1.099.000`.

---

## Penyimpanan Data

`DATA_DIR` berisi dua file JSON yang ditulis secara atomik:

- `dataset.json`: hasil impor terakhir yang sudah dikonfirmasi.
- `config.json`: pemetaan status, pemetaan nama produk → SKU, dan nama tampilan produk. Tetap ada di antara unggahan.

Impor yang belum dikonfirmasi hanya disimpan di memori dan hilang jika aplikasi restart.

---

## Deployment

Produksi berjalan di Dokploy: **https://biota.azis.works** (project `biota`, environment `production`, volume `biota-data` di `/data`).

- **Pull request ke `main`** menjalankan `.github/workflows/pr.yml`: generate templ + Tailwind, `go build`, `go vet`, golangci-lint, `go test`.
- **Publish GitHub Release** menjalankan `.github/workflows/release.yml`: test → build dan push `ghcr.io/azisuazusa/biota:<tag>` (linux/amd64) → arahkan aplikasi Dokploy ke tag tersebut → deploy.
  - Hanya tag stabil `vX.Y.Z` yang di-deploy. Prerelease dan format tag lain dilewati.
  - Tanpa approval manual: publish release sudah dianggap sebagai approval.

```sh
gh release create v1.2.3 --title "v1.2.3" --generate-notes
```

**Rollback:** buka run Actions dari release lama, lalu pilih "Re-run all jobs". Dokploy akan kembali memakai image tag lama tersebut.

**Secret repo** (nama saja): `DOKPLOY_URL`, `DOKPLOY_API_KEY`, `DOKPLOY_APPLICATION_ID`.

Package GHCR `ghcr.io/azisuazusa/biota` harus **public** agar Dokploy bisa menarik image tanpa kredensial.

---

## Struktur Proyek

- `services/sales/`: domain (entitas, normalisasi, impor/review, analitik, port). Tanpa HTTP atau akses file.
- `services/sales/repo_file/`: adapter file JSON untuk port.
- `app/http/controller/sales/`: handler chi.
- `app/http/views/`: view templ.
- `app/router.go` (rute), `app/http/main.go` (entrypoint).
- `test_data/`: fixture CSV (`sample_orders_oct2026.csv` adalah data sintetis untuk pengujian).
