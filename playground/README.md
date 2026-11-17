# Playground Fase 0

Dua latihan kecil sebelum menyentuh kode project, sesuai roadmap Fase 0. Folder ini module Go tersendiri, jadi `make test` dan CI di root tidak menjalankannya.

| Latihan | Aturan di PRD | File yang diisi |
| --- | --- | --- |
| Kode booking | BR-10: 6 karakter, tanpa 0, O, 1, I | `bookingcode/bookingcode.go` |
| Harga pesanan | BR-02, BR-04, BR-05 | `pricing/pricing.go` |

## Cara mengerjakan

```sh
cd playground
go test ./...         # merah: fungsinya masih TODO
# isi TODO di bookingcode.go dan pricing.go
go test -race ./...   # ulangi sampai hijau
```

Test sudah ditulis lebih dulu, gaya Learn Go with Tests. Baca test-nya untuk memahami kebutuhan, lalu tulis kode secukupnya sampai lulus. Setelah hijau, tambahkan minimal satu kasus test buatanmu sendiri di `pricing_test.go`.

## Yang dilatih

- table-driven test dengan `t.Run`
- error sebagai nilai: `errors.New` dan `errors.Is`
- `crypto/rand` dibanding `math/rand`, dan kenapa alfabet 32 karakter membuat setiap karakter berpeluang sama
- uang sebagai `int64` dan urutan operasi pembagian integer: untuk tarif 999.999, `Base*75/100` menghasilkan 749.999, sedangkan `Base/100*75` hanya 749.925

Di Fase 3, kedua fungsi ini pindah ke `internal/booking` (lihat tab 07).
