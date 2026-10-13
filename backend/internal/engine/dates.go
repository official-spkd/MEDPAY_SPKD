// Package engine berisi logika bisnis murni MedPay (tanpa HTTP & tanpa database):
// rantai tarif & gap, scrubber 14 aturan, deadline, KAPJ, aging/DSO,
// pencocokan mutasi bank, grouper demo, dan transisi status klaim.
// Semua fungsi deterministik sehingga mudah diuji (lihat *_test.go).
package engine

import (
	"math"
	"time"
)

// loc adalah zona waktu aplikasi; "hari ini" dan tanggal kalender dihitung di zona ini.
var loc = time.FixedZone("WIB", 7*60*60)

// SetLocation mengganti zona waktu aplikasi (dipanggil sekali saat startup).
func SetLocation(l *time.Location) {
	if l != nil {
		loc = l
	}
}

// Location mengembalikan zona waktu aplikasi.
func Location() *time.Location { return loc }

// Day mengubah waktu menjadi tanggal kalender di zona aplikasi (00:00 UTC),
// sehingga selisih hari tidak terpengaruh jam atau DST.
func Day(t time.Time) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Today adalah tanggal hari ini di zona aplikasi.
func Today() time.Time { return Day(time.Now()) }

// DaysBetween = jumlah hari kalender dari a ke b (negatif bila b sebelum a).
func DaysBetween(a, b time.Time) int {
	return int(math.Round(Day(b).Sub(Day(a)).Hours() / 24))
}

// LOS = lama rawat (hari) dari tanggal masuk ke tanggal keluar, minimal 0.
func LOS(admission, discharge *time.Time) int {
	if admission == nil || discharge == nil {
		return 0
	}
	if n := DaysBetween(*admission, *discharge); n > 0 {
		return n
	}
	return 0
}
