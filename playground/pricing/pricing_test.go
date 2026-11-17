package pricing

import (
	"errors"
	"testing"
)

func TestCalculate(t *testing.T) {
	cgkDps := Fare{Base: 1_100_000, Tax: 137_500} // contoh CGK → DPS di tab 05

	tests := []struct {
		name    string
		pax     Pax
		fare    Fare
		want    int64
		wantErr error
	}{
		{
			name: "2 dewasa, sama dengan contoh respons API",
			pax:  Pax{Adults: 2},
			fare: cgkDps,
			want: 2_480_000, // 2 × (1.100.000 + 137.500) + 5.000
		},
		{
			name: "1 dewasa, 1 anak, 1 bayi",
			pax:  Pax{Adults: 1, Children: 1, Infants: 1},
			fare: cgkDps,
			want: 2_315_000, // 1.237.500 + (825.000 + 137.500) + 110.000 + 5.000
		},
		{
			name: "persentase dibulatkan ke bawah",
			pax:  Pax{Adults: 1, Children: 1, Infants: 1},
			fare: Fare{Base: 999_999, Tax: 100_000},
			want: 2_054_997, // anak 749.999 (bukan 749.999,25), bayi 99.999 (bukan 99.999,9)
		},
		{
			name: "7 kursi, batas maksimum",
			pax:  Pax{Adults: 4, Children: 3},
			fare: Fare{Base: 500_000, Tax: 50_000},
			want: 3_480_000,
		},
		// TODO(latihan): tambahkan kasus lain yang menurutmu penting.
		{name: "tanpa dewasa ditolak", pax: Pax{Children: 1}, fare: cgkDps, wantErr: ErrInvalidPax},
		{name: "8 kursi ditolak", pax: Pax{Adults: 5, Children: 3}, fare: cgkDps, wantErr: ErrInvalidPax},
		{name: "bayi lebih banyak dari dewasa ditolak", pax: Pax{Adults: 2, Infants: 3}, fare: cgkDps, wantErr: ErrInvalidPax},
		{name: "jumlah negatif ditolak", pax: Pax{Adults: 1, Children: -1}, fare: cgkDps, wantErr: ErrInvalidPax},
		{name: "tarif dasar nol ditolak", pax: Pax{Adults: 1}, fare: Fare{Base: 0, Tax: 10_000}, wantErr: ErrInvalidFare},
		{name: "pajak negatif ditolak", pax: Pax{Adults: 1}, fare: Fare{Base: 100_000, Tax: -1}, wantErr: ErrInvalidFare},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.pax, tt.fare)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Calculate(%+v, %+v) error = %v, want %v", tt.pax, tt.fare, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Calculate(%+v, %+v) unexpected error: %v", tt.pax, tt.fare, err)
			}
			if got != tt.want {
				t.Errorf("Calculate(%+v, %+v) = %d, want %d", tt.pax, tt.fare, got, tt.want)
			}
		})
	}
}
