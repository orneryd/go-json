package json_test

import (
	"bytes"
	std "encoding/json"
	"fmt"
	json "github.com/orneryd/go-json"
	"io"
	"reflect"
	"strings"
	"testing"
)

type stringMapReferenceMap map[string]string

func stringMapErrorSignature(err error) string {
	if err == nil {
		return "nil"
	}
	s := fmt.Sprintf("%T:%s", err, err.Error())
	v := reflect.Indirect(reflect.ValueOf(err))
	if v.IsValid() && v.Kind() == reflect.Struct {
		for _, n := range []string{"Offset", "Value", "Struct", "Field"} {
			f := v.FieldByName(n)
			if f.IsValid() && f.CanInterface() {
				s += fmt.Sprintf("|%s=%v", n, f.Interface())
			}
		}
	}
	return strings.ReplaceAll(s, "json_test.stringMapReferenceMap", "map[string]string")
}
func stringMapCorpus() []string {
	c := []string{"null", " null ", "{}", " { } ", `{"a":"one","b":"two"}`, `{"a":"one","a":"two"}`, `{"a":null}`, `{"a":"one","b":3,"c":"three"}`, `{"a":true,"b":"two"}`, `{"a":[1,2],"b":"two"}`, `{"a":{"x":1},"b":"two"}`, `{"a":"one","b":"two",}`, `{"a":"one"`, `{"a":"one"} trailing`, `{"a":"one"} {}`, `{"a":"bad\q"}`, `{null:"value"}`, `{4:"value"}`, `[]`, `"text"`, `false`, `1`, "", `{"a":"one","b":`, "{\"a\":\"x\x00y\"}", `{"\ud800":"\udfff","pair":"\ud83d\ude00"}`, `{"invalid":"` + string([]byte{0xff, 0xed, 0xa0, 0x80}) + `"}`}
	for _, k := range []string{"a", "é", `\ud800`, "comma,colon:", "quoted\\\"key"} {
		for _, v := range []string{`"text"`, `null`, `true`, `42`, `[]`, `{}`, `"\ud800"`, `"\u0000"`, `"\\literal"`} {
			c = append(c, `{"`+k+`":`+v+`,"tail":"valid"}`)
		}
	}
	for _, n := range []int{1, 20, 100, 10001} {
		c = append(c, `{"nested":`+strings.Repeat("[", n)+`0`+strings.Repeat("]", n)+`,"tail":"valid"}`)
	}
	return c
}
func TestStringMapDifferential(t *testing.T) {
	for _, preserve := range []bool{false, true} {
		for _, reuse := range []bool{false, true} {
			for i, input := range stringMapCorpus() {
				t.Run(fmt.Sprintf("preserve=%t/reuse=%t/case=%d", preserve, reuse, i), func(t *testing.T) {
					var got map[string]string
					var ref stringMapReferenceMap
					if reuse {
						got = map[string]string{"keep": "seed", "a": "old"}
						ref = stringMapReferenceMap{"keep": "seed", "a": "old"}
					}
					var opts []json.DecodeOptionFunc
					if preserve {
						opts = append(opts, json.DecodePreserveInvalidUTF8())
					}
					err := json.UnmarshalWithOption([]byte(input), &got, opts...)
					wantErr := json.UnmarshalWithOption([]byte(input), &ref, opts...)
					if !reflect.DeepEqual(got, map[string]string(ref)) || stringMapErrorSignature(err) != stringMapErrorSignature(wantErr) {
						t.Fatalf("input=%q got=%#v %s generic=%#v %s", input, got, stringMapErrorSignature(err), ref, stringMapErrorSignature(wantErr))
					}
					if !preserve && std.Valid([]byte(input)) {
						var want map[string]string
						if reuse {
							want = map[string]string{"keep": "seed", "a": "old"}
						}
						se := std.Unmarshal([]byte(input), &want)
						if !reflect.DeepEqual(got, want) || (err == nil) != (se == nil) {
							t.Fatalf("std input=%q got=%#v %v want=%#v %v", input, got, err, want, se)
						}
					}
				})
			}
		}
	}
}

type stringMapNamedKey string

func (k *stringMapNamedKey) UnmarshalText(raw []byte) error {
	*k = stringMapNamedKey("key:" + string(raw))
	return nil
}

type stringMapNamedValue string

func (v *stringMapNamedValue) UnmarshalJSON(raw []byte) error {
	var s string
	if err := std.Unmarshal(raw, &s); err != nil {
		return err
	}
	*v = stringMapNamedValue("value:" + s)
	return nil
}
func TestStringMapDispatch(t *testing.T) {
	var k map[stringMapNamedKey]string
	if err := json.Unmarshal([]byte(`{"x":"y"}`), &k); err != nil || k["key:x"] != "y" {
		t.Fatalf("key hook %#v %v", k, err)
	}
	var v map[string]stringMapNamedValue
	if err := json.Unmarshal([]byte(`{"x":"y"}`), &v); err != nil || v["x"] != "value:y" {
		t.Fatalf("value hook %#v %v", v, err)
	}
	type alias = map[string]string
	var a alias
	if err := json.Unmarshal([]byte(`{"x":"y"}`), &a); err != nil || a["x"] != "y" {
		t.Fatalf("alias %#v %v", a, err)
	}
}

func FuzzStringMapDifferential(f *testing.F) {
	for _, input := range stringMapCorpus() {
		f.Add([]byte(input))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		for _, preserve := range []bool{false, true} {
			got := map[string]string{"keep": "seed", "a": "old"}
			ref := stringMapReferenceMap{"keep": "seed", "a": "old"}
			var options []json.DecodeOptionFunc
			if preserve {
				options = append(options, json.DecodePreserveInvalidUTF8())
			}
			err := json.UnmarshalWithOption(raw, &got, options...)
			expected := json.UnmarshalWithOption(raw, &ref, options...)
			if !reflect.DeepEqual(got, map[string]string(ref)) || stringMapErrorSignature(err) != stringMapErrorSignature(expected) {
				t.Fatalf("input=%q exact=%#v %s generic=%#v %s", raw, got, stringMapErrorSignature(err), ref, stringMapErrorSignature(expected))
			}
		}
	})
}

type stringMapErrorPathInner struct {
	Values map[string]string `json:"values"`
}
type stringMapErrorPathOuter struct {
	Items []stringMapErrorPathInner `json:"items"`
}

func TestStringMapNestedTypeErrorPath(t *testing.T) {
	for _, input := range []string{
		`{"items":[{"values":{"first":"ok","bad":42,"later":"ok"}}]}`,
		`{"items":[{"values":{"first":"ok"}},{"values":{"escaped\u006bey":true,"later":"ok"}}]}`,
		`{"items":[{"values":{"nested":{"value":"wrong"},"later":"ok"}}]}`,
	} {
		var got, want stringMapErrorPathOuter
		err := json.Unmarshal([]byte(input), &got)
		expected := std.Unmarshal([]byte(input), &want)
		a, ok := err.(*json.UnmarshalTypeError)
		if !ok {
			t.Fatalf("expected typed dependency error, got %T %v", err, err)
		}
		b, ok := expected.(*std.UnmarshalTypeError)
		if !ok {
			t.Fatalf("expected typed std error, got %T %v", expected, expected)
		}
		if a.Value != b.Value || a.Type != b.Type || a.Offset != b.Offset || a.Struct != b.Struct || a.Field != b.Field || !reflect.DeepEqual(got, want) {
			t.Fatalf("input=%s dependency=%+v values=%#v standard=%+v values=%#v", input, a, got, b, want)
		}
		t.Logf("input=%s Type=%s Value=%s Offset=%d Struct=%s Field=%s", input, a.Type, a.Value, a.Offset, a.Struct, a.Field)
	}
}

func stringMapDecodePublicAPI[T any](api string, raw []byte, out *T) error {
	switch api {
	case "Unmarshal":
		return json.Unmarshal(raw, out)
	case "UnmarshalWithOption":
		return json.UnmarshalWithOption(raw, out, json.DecodePreserveInvalidUTF8())
	case "UnmarshalOf":
		return json.UnmarshalOf(raw, out)
	case "Decoder":
		return json.NewDecoder(bytes.NewReader(raw)).Decode(out)
	default:
		panic("unknown API")
	}
}
func stringMapAssertCompleteTypeError(t *testing.T, got, want error) {
	t.Helper()
	if got == nil && want == nil {
		return
	}
	a, ok := got.(*json.UnmarshalTypeError)
	if !ok {
		t.Fatalf("dependency error %T %v, wanted %T %v", got, got, want, want)
	}
	b, ok := want.(*std.UnmarshalTypeError)
	if !ok {
		t.Fatalf("standard error %T %v", want, want)
	}
	if a.Value != b.Value || a.Type != b.Type || a.Offset != b.Offset || a.Struct != b.Struct || a.Field != b.Field {
		t.Fatalf("complete type error mismatch dependency=%+v standard=%+v", a, b)
	}
}
func TestStringMapPublicAPIErrorParity(t *testing.T) {
	type alias = map[string]string
	for _, api := range []string{"Unmarshal", "UnmarshalWithOption", "UnmarshalOf", "Decoder"} {
		for _, input := range []string{`{"a":3}`, `{"before":"ok","a":3,"after":"kept"}`, `{"escaped\u006bey":true}`} {
			t.Run(api+"/root/"+input, func(t *testing.T) {
				var got map[string]string
				var aliased alias
				var named stringMapReferenceMap
				var want map[string]string
				err := stringMapDecodePublicAPI(api, []byte(input), &got)
				aliasErr := stringMapDecodePublicAPI(api, []byte(input), &aliased)
				namedErr := stringMapDecodePublicAPI(api, []byte(input), &named)
				var standard error
				if api == "Decoder" {
					standard = std.NewDecoder(strings.NewReader(input)).Decode(&want)
				} else {
					standard = std.Unmarshal([]byte(input), &want)
				}
				stringMapAssertCompleteTypeError(t, err, standard)
				stringMapAssertCompleteTypeError(t, aliasErr, standard)
				stringMapAssertCompleteTypeError(t, namedErr, standard)
				if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(aliased, want) || !reflect.DeepEqual(map[string]string(named), want) {
					t.Fatalf("partial values exact=%#v alias=%#v named=%#v standard=%#v", got, aliased, named, want)
				}
				t.Logf("%s root Field=%q Offset=%d values=%#v", api, err.(*json.UnmarshalTypeError).Field, err.(*json.UnmarshalTypeError).Offset, got)
			})
		}
		t.Run(api+"/nested", func(t *testing.T) {
			input := `{"items":[{"values":{"ok":"before","a":3,"after":"kept"}}]}`
			var got, want stringMapErrorPathOuter
			err := stringMapDecodePublicAPI(api, []byte(input), &got)
			var standard error
			if api == "Decoder" {
				standard = std.NewDecoder(strings.NewReader(input)).Decode(&want)
			} else {
				standard = std.Unmarshal([]byte(input), &want)
			}
			stringMapAssertCompleteTypeError(t, err, standard)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("partial nested values got=%#v standard=%#v", got, want)
			}
		})
	}
}

func TestStringMapSequentialStreamReuse(t *testing.T) {
	input := `{"first":"one"} {"bad":3,"after":"two"} null {} {"last":"three"}`
	gotDecoder := json.NewDecoder(strings.NewReader(input))
	wantDecoder := std.NewDecoder(strings.NewReader(input))
	var got, want map[string]string
	var first map[string]string
	for i := 0; i < 5; i++ {
		err := gotDecoder.Decode(&got)
		standard := wantDecoder.Decode(&want)
		stringMapAssertCompleteTypeError(t, err, standard)
		if !reflect.DeepEqual(got, want) || gotDecoder.InputOffset() != wantDecoder.InputOffset() {
			t.Fatalf("step%d got=%#v err=%v offset=%d want=%#v err=%v offset=%d", i, got, err, gotDecoder.InputOffset(), want, standard, wantDecoder.InputOffset())
		}
		if i == 0 {
			first = got
		}
		if i == 2 && got != nil {
			t.Fatal("null did not clear current map")
		}
	}
	if first["first"] != "one" || first["after"] != "two" || first["bad"] != "" || len(first) != 3 {
		t.Fatalf("retained prior stream map changed or lost partial entries: %#v", first)
	}
	if err := gotDecoder.Decode(&got); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
	// A syntax error remains sticky, matching the established decoder contract.
	bad := `{"ok":"one"} {"bad":`
	a := json.NewDecoder(strings.NewReader(bad))
	b := std.NewDecoder(strings.NewReader(bad))
	for i := 0; i < 3; i++ {
		var x, y map[string]string
		e := a.Decode(&x)
		w := b.Decode(&y)
		if (e == nil) != (w == nil) || !reflect.DeepEqual(x, y) {
			t.Fatalf("syntax step%d got=%#v %v standard=%#v %v", i, x, e, y, w)
		}
		if i > 0 && (e != io.ErrUnexpectedEOF || w != io.ErrUnexpectedEOF) {
			t.Fatalf("truncated stream must retain UnexpectedEOF: dependency=%T %v standard=%T %v", e, e, w, w)
		}
	}
}

func TestStringMapDepthBoundary(t *testing.T) {
	for _, arrays := range []int{9999, 10000} {
		raw := []byte(`{"value":` + strings.Repeat("[", arrays) + `0` + strings.Repeat("]", arrays) + `}`)
		var got map[string]string
		var ref stringMapReferenceMap
		var want map[string]string
		e := json.Unmarshal(raw, &got)
		r := json.Unmarshal(raw, &ref)
		w := std.Unmarshal(raw, &want)
		if stringMapErrorSignature(e) != stringMapErrorSignature(r) || !reflect.DeepEqual(got, map[string]string(ref)) {
			t.Fatalf("depth%d exact=%#v %v generic=%#v %v", arrays, got, e, ref, r)
		}
		if (e == nil) != (w == nil) {
			t.Fatalf("depth%d standard acceptance differs: %v versus %v", arrays, e, w)
		}
		if arrays == 9999 {
			stringMapAssertCompleteTypeError(t, e, w)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("valid-depth partial values differ: dependency=%#v standard=%#v", got, want)
			}
		} else {
			// Established pinned generic behavior rejects total depth10001 with a
			// type error, while encoding/json rejects it with a syntax error before
			// binding. This specialization preserves the existing decoder contract;
			// no new acceptance or security impact has been established.
			if _, ok := e.(*json.UnmarshalTypeError); !ok {
				t.Fatalf("pinned generic error class changed: %T %v", e, e)
			}
			if _, ok := w.(*std.SyntaxError); !ok {
				t.Fatalf("standard over-depth error changed: %T %v", w, w)
			}
			t.Logf("documented baseline boundary: dependency=%T %v standard=%T %v", e, e, w, w)
		}
	}
}
