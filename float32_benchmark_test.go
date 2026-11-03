package json_test

import (
	"testing"

	"github.com/orneryd/go-json"
)

func BenchmarkFloat32DestinationBinding(b *testing.B) {
	for _, token := range []string{"1.25", "1.1", "3.4028235e38", "1.0000000596046448"} {
		b.Run(token, func(b *testing.B) {
			raw := []byte(token)
			var value float32
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// The historical arm rejects the valid maximum and rounds the
				// midpoint incorrectly. Those rows compare cost, not correctness.
				_ = json.Unmarshal(raw, &value)
			}
		})
	}
}
