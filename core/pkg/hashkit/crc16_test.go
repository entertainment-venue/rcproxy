package hashkit

import (
	"testing"
)

func Test_Crc16(t *testing.T) {
	if v := Hash("jiofiejjkeofijo"); v != 14761 {
		t.Fatalf("crc16 hash error, need: %d got: %d", 14761, v)
	}

	if v := Hash(""); v != 0 {
		t.Fatalf("crc16 hash error, need: %d got: %d", 0, v)
	}
}

func BenchmarkCrc16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Hash("jiofiejjkeofijo")
	}
}

func Test_Crc16HashTag(t *testing.T) {
	if v := Hash("{jio}fiejjkeofijo"); v != 12369 {
		t.Fatalf("crc16 hash tag error, need: %d got: %d", 12369, v)
	}
	if v := Hash("jioj{jio}fiejjkeofijo"); v != 12369 {
		t.Fatalf("crc16 hash tag error, need: %d got: %d", 12369, v)
	}
	if v := Hash("fiejjkeofijo{jio}"); v != 12369 {
		t.Fatalf("crc16 hash tag error, need: %d got: %d", 12369, v)
	}
	if v := Hash("fiejjkeofijo{jio}{abc}"); v != 12369 {
		t.Fatalf("crc16 hash tag error, need: %d got: %d", 12369, v)
	}
}

func BenchmarkCrc16Hasher_Hash(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash("jioj{jio}fiejjkeofijo")
	}
}