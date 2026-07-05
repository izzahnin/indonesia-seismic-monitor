package mapper

import "testing"

func TestMapToProvince(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lng  float64
		want string
	}{
		{"aceh", 4.0, 96.0, "Aceh"},
		{"jawa timur", -7.5, 112.5, "Jawa Timur"},
		{"sulawesi utara", 1.5, 125.0, "Sulawesi Utara"},
		{"laut terbuka", 0.0, 110.0, "Wilayah Lain"},
		{"di luar indonesia", 35.0, 139.0, "Wilayah Lain"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MapToProvince(tc.lat, tc.lng)
			if got != tc.want {
				t.Errorf("MapToProvince(%v, %v) = %q, want %q", tc.lat, tc.lng, got, tc.want)
			}
		})
	}
}
