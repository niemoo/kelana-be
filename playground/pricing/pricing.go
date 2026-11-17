// Package pricing adalah latihan Fase 0: menghitung total harga pesanan
// menurut BR-02 (jumlah penumpang), BR-04 (harga anak dan bayi), dan BR-05
// (biaya layanan). Semua uang dalam rupiah bertipe int64, tidak pernah float.
package pricing

import "errors"

// ServiceFee adalah biaya layanan per pesanan (BR-05).
const ServiceFee int64 = 5_000

// Pax adalah jumlah penumpang per tipe (BR-01).
type Pax struct {
	Adults   int // dewasa, 12 tahun ke atas
	Children int // anak, 2–11 tahun
	Infants  int // bayi, di bawah 2 tahun; tidak menempati kursi
}

// Fare adalah tarif satu penumpang dewasa dari maskapai.
type Fare struct {
	Base int64 // tarif dasar
	Tax  int64 // pajak dan iuran bandara
}

// ErrInvalidPax dikembalikan bila jumlah penumpang melanggar BR-02.
var ErrInvalidPax = errors.New("pricing: invalid passenger count")

// ErrInvalidFare dikembalikan bila Base tidak lebih dari nol atau Tax negatif.
var ErrInvalidFare = errors.New("pricing: invalid fare")

// ErrNotImplemented dikembalikan selama latihan ini belum dikerjakan.
var ErrNotImplemented = errors.New("pricing: not implemented yet")

// Calculate menghitung total harga pesanan:
//
//   - dewasa: Base + Tax
//   - anak: 75% dari Base, ditambah Tax penuh
//   - bayi: 10% dari Base, tanpa Tax
//   - ditambah ServiceFee sekali per pesanan
//
// Persentase dibulatkan ke bawah ke rupiah penuh, cukup dengan pembagian
// integer (kalikan dulu, baru bagi).
//
// Jumlah penumpang harus memenuhi BR-02: minimal 1 dewasa, dewasa + anak
// paling banyak 7, bayi tidak lebih banyak dari dewasa, dan tidak ada jumlah
// negatif. Bila dilanggar, kembalikan ErrInvalidPax; bila tarifnya tidak
// valid, kembalikan ErrInvalidFare.
func Calculate(p Pax, f Fare) (int64, error) {
	// TODO(latihan): hapus baris di bawah dan tulis implementasinya,
	// sampai `go test ./pricing` hijau.
	
	if p.Adults < 1 || p.Children < 0 || p.Infants < 0 || p.Adults + p.Children > 7 || p.Infants > p.Adults {
		return 0, ErrInvalidPax
	}

	if f.Base <= 0 || f.Tax < 0 {
		return 0, ErrInvalidFare
	}

	adult := f.Base + f.Tax
	child := (f.Base * 75 / 100) + f.Tax
	infant := f.Base * 10 / 100

	total := adult * int64(p.Adults) + child * int64(p.Children) + infant * int64(p.Infants) + ServiceFee

	return total, nil
}
