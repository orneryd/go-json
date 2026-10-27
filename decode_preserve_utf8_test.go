package json_test

import (
	"encoding/json"
	"fmt"
	fork "github.com/orneryd/go-json"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestDecodePreserveInvalidUTF8(t *testing.T) {
	high, low := "\xed\xa0\x80", "\xed\xb0\x80"
	for _, raw := range []string{high, low, "\xff", "ordinary", "日本", high + `\n` + low, `\ud800`, `\ud83d\ude00`} {
		input := []byte(`"` + raw + `"`)
		var normal string
		if err := json.Unmarshal(input, &normal); err != nil {
			t.Fatal(err)
		}
		for _, preserve := range []bool{false, true, false} {
			var opts []fork.DecodeOptionFunc
			want := normal
			if preserve {
				opts = append(opts, fork.DecodePreserveInvalidUTF8())
				if strings.ContainsAny(raw, "\xed\xff") {
					want = strings.ReplaceAll(raw, `\n`, "\n")
				}
			}
			var got string
			if err := fork.UnmarshalWithOption(input, &got, opts...); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("preserve=%v input=%q got=%q want=%q", preserve, raw, got, want)
			}
		}
	}
	input := []byte(`{"` + high + `":"` + low + `","` + low + `":"` + high + `"}`)
	var got map[string]string
	if err := fork.UnmarshalWithOption(input, &got, fork.DecodePreserveInvalidUTF8()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[string]string{high: low, low: high}) {
		t.Fatalf("map identity: %#v", got)
	}
	var normal map[string]string
	if err := json.Unmarshal(input, &normal); err != nil {
		t.Fatal(err)
	}
	if err := fork.Unmarshal(input, &got); err != nil {
		t.Fatal(err)
	}
	// Both decoders preserve pre-existing entries; compare newly decoded maps.
	var actual map[string]string
	if err := fork.Unmarshal(input, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, normal) {
		t.Fatalf("default changed: %#v want %#v", actual, normal)
	}
	for _, bad := range []string{`"unterminated`, "\"line\nnext\"", `"\q"`} {
		var value string
		if fork.UnmarshalWithOption([]byte(bad), &value, fork.DecodePreserveInvalidUTF8()) == nil {
			t.Fatalf("accepted malformed input %q", bad)
		}
	}
}

func TestDecodePreserveInvalidUTF8LongStreamAndPath(t *testing.T) {
	high, low := "\xed\xa0\x80", "\xed\xb0\x80"
	for _, size := range []int{0, 511, 513, 65535, 65537} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			pad := strings.Repeat("x", size)
			input := []byte(`"` + pad + high + `\n` + low + `"`)
			want := pad + high + "\n" + low
			var value string
			if err := fork.UnmarshalWithOption(input, &value, fork.DecodePreserveInvalidUTF8()); err != nil {
				t.Fatal(err)
			}
			if value != want {
				t.Fatal("long scalar changed")
			}
			var field struct {
				Name string `json:"name"`
			}
			if err := fork.UnmarshalWithOption([]byte(`{"name":`+string(input)+`}`), &field, fork.DecodePreserveInvalidUTF8()); err != nil {
				t.Fatal(err)
			}
			if field.Name != want {
				t.Fatal("long field changed")
			}
			decoder := fork.NewDecoder(&losslessByteReader{input})
			if err := decoder.DecodeWithOption(&value, fork.DecodePreserveInvalidUTF8()); err != nil {
				t.Fatal(err)
			}
			if value != want {
				t.Fatal("stream changed")
			}
			if err := decoder.Decode(&value); err != io.EOF {
				t.Fatalf("stream end: %v", err)
			}
		})
	}
	path, err := fork.CreatePath("$.name")
	if err != nil {
		t.Fatal(err)
	}
	var selected string
	if err := path.Unmarshal([]byte(`{"name":"`+high+`"}`), &selected, fork.DecodePreserveInvalidUTF8()); err != nil {
		t.Fatal(err)
	}
	if selected != high {
		t.Fatalf("path lost bytes: %q", selected)
	}
	decoder := fork.NewDecoder(strings.NewReader(`null ["` + high + `","` + low + `"]`))
	var header any
	if err := decoder.DecodeWithOption(&header, fork.DecodePreserveInvalidUTF8()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []any{json.Delim('['), high, low, json.Delim(']')} {
		got, err := decoder.Token()
		if err != nil || got != want {
			t.Fatalf("token %#v/%v want %#v", got, err, want)
		}
	}
}

type losslessByteReader struct{ raw []byte }

func (r *losslessByteReader) Read(out []byte) (int, error) {
	if len(r.raw) == 0 {
		return 0, io.EOF
	}
	if len(out) == 0 {
		return 0, nil
	}
	out[0] = r.raw[0]
	r.raw = r.raw[1:]
	return 1, nil
}

func TestDecodePreserveInvalidUTF8ConcurrentIsolation(t *testing.T) {
	input := []byte(`"` + "\xed\xa0\x80" + `"`)
	var normal string
	if err := json.Unmarshal(input, &normal); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for n := 0; n < 12; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 64; j++ {
				var got string
				var opts []fork.DecodeOptionFunc
				want := normal
				if (n+j)%2 == 0 {
					opts = append(opts, fork.DecodePreserveInvalidUTF8())
					want = "\xed\xa0\x80"
				}
				if err := fork.UnmarshalWithOption(input, &got, opts...); err != nil {
					t.Error(err)
					return
				}
				if got != want {
					t.Errorf("option leaked: got=%q want=%q", got, want)
					return
				}
			}
		}(n)
	}
	wg.Wait()
}
