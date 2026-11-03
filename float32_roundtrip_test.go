package json_test

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/orneryd/go-json"
)

func TestFloat32DecimalBindingMatchesStandard(t *testing.T) {
	for _, token := range []string{
		"3.4028235e38", "-3.4028235e38", "3.4028234663852886e38",
		"3.40282356e38", "3.40282357e38", "-3.40282357e38", "1e400",
		"1.0000000596046448", "-1.0000000596046448", "1.40129846e-45",
		"7.006492321624085e-46", "1e-50", "-0", "0", "1.25", "null",
	} {
		t.Run(token, func(t *testing.T) {
			type record struct {
				Value float32
				After int
			}
			data := []byte(`{"Value":` + token + `,"After":7}`)
			var want record
			wantErr := stdjson.Unmarshal(data, &want)
			for _, stream := range []bool{false, true} {
				var got record
				var err error
				if stream {
					err = json.NewDecoder(bytes.NewReader(data)).Decode(&got)
				} else {
					err = json.Unmarshal(data, &got)
				}
				if (err == nil) != (wantErr == nil) {
					t.Fatalf("stream=%v: error=%v, standard=%v", stream, err, wantErr)
				}
				if err == nil && (got != want || math.Float32bits(got.Value) != math.Float32bits(want.Value)) {
					t.Fatalf("stream=%v: got=%+v/%08x want=%+v/%08x", stream, got, math.Float32bits(got.Value), want, math.Float32bits(want.Value))
				}
				if wantErr != nil {
					var typed *json.UnmarshalTypeError
					if !errors.As(err, &typed) || typed.Type != reflect.TypeOf(float32(0)) {
						t.Fatalf("expected float32 type error, got %T: %v", err, err)
					}
					if got.After != 7 {
						t.Fatalf("type error prevented later field binding: %+v", got)
					}
				}
			}
		})
	}
}

func TestFloat32MarshalRoundTripExtrema(t *testing.T) {
	for _, value := range []float32{math.MaxFloat32, -math.MaxFloat32, math.SmallestNonzeroFloat32, -math.SmallestNonzeroFloat32, float32(math.Copysign(0, -1))} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var got float32
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if math.Float32bits(value) != math.Float32bits(got) {
			t.Fatalf("%s: got=%v want=%v", raw, got, value)
		}
	}
	decoder := json.NewDecoder(strings.NewReader("3.4028235e38 1.25"))
	for _, want := range []float32{math.MaxFloat32, 1.25} {
		var got float32
		if err := decoder.Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("got=%v want=%v", got, want)
		}
	}
}
