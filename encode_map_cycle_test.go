package json_test

import (
	"bytes"
	std "encoding/json"
	"errors"
	json "github.com/orneryd/go-json"
	"testing"
)

func TestEncodeMapCycleIdentity(t *testing.T) {
	direct := map[string]any{}
	direct["self"] = direct
	left, right := map[string]any{}, map[string]any{}
	left["next"] = right
	right["next"] = left
	mixed := map[string]any{}
	mixed["items"] = []any{mixed}
	for _, options := range [][]json.EncodeOptionFunc{nil, {json.DisableNormalizeUTF8()}, {json.DisableHTMLEscape()}, {json.UnorderedMap()}} {
		for _, cycle := range []any{direct, left, mixed} {
			_, err := json.MarshalWithOption(cycle, options...)
			var unsupported *json.UnsupportedValueError
			if !errors.As(err, &unsupported) {
				t.Fatalf("cycle error: %T %v", err, err)
			}
		}
	}
	shared := map[string]any{"name": "shared"}
	value := nestedInterfaceValue(1100, []any{shared, shared})
	expected, err := std.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expected, got) {
		t.Fatal("shared acyclic map changed")
	}
}
