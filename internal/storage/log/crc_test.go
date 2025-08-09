package log

import "testing"

func TestComputeCRC32C_KnownVectors(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want uint32
	}{
		{"empty", []byte(""), 0x00000000},
		{"123456789", []byte("123456789"), 0xE3069283},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeCRC32C(tt.in)
			if got != tt.want {
				t.Fatalf("ComputeCRC32C(%q) = 0x%08X, want 0x%08X", tt.in, got, tt.want)
			}
		})
	}
}
