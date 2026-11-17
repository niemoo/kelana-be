// Package bookingcode adalah latihan Fase 0: membuat kode booking sesuai
// BR-10, yaitu 6 karakter huruf besar dan angka tanpa 0, O, 1, dan I.
package bookingcode

import "crypto/rand"

// Alphabet berisi karakter yang boleh dipakai: A–Z tanpa I dan O, lalu 2–9.
const Alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// Length adalah panjang kode booking.
const Length = 6

// Generate membuat kode booking acak sepanjang Length dari Alphabet.
//
// Petunjuk:
//   - Pakai crypto/rand, bukan math/rand: kode booking dipakai untuk membuka
//     pesanan, jadi harus sulit ditebak.
//   - Sejak Go 1.24, crypto/rand.Read selalu mengisi penuh slice-nya dan tidak
//     pernah mengembalikan error.
//   - Alphabet berisi tepat 32 karakter. Pikirkan mengapa angka 32 membuat
//     setiap karakter punya peluang yang sama saat dipilih dari satu byte acak.
func Generate() string {
	// TODO(latihan): hapus baris di bawah dan tulis implementasinya,
	// sampai `go test ./bookingcode` hijau.

	code := make([]byte, Length)
	rand.Read(code)

	for i := range code {
		code[i] = Alphabet[code[i] % byte(len(Alphabet))]
	}

	return string(code)
}
