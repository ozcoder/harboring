package har

import (
	"strings"
	"testing"
)

func TestDecodeOrderedKeyOrder(t *testing.T) {
	data := []byte(`{"z":1,"a":2,"m":3,"b":4}`)
	v, err := decodeOrdered(data)
	if err != nil {
		t.Fatal(err)
	}

	if v.typ != jsonObject {
		t.Fatal("expected object")
	}

	expected := []string{"z", "a", "m", "b"}
	if len(v.keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d: %v", len(expected), len(v.keys), v.keys)
	}
	for i, k := range expected {
		if v.keys[i] != k {
			t.Fatalf("key[%d]: expected %q, got %q", i, k, v.keys[i])
		}
	}
}

func TestMarshalOrderedPreservesOrder(t *testing.T) {
	v := &orderedValue{typ: jsonObject, fields: make(map[string]*orderedValue)}
	v.keys = []string{"z", "a", "m", "b"}
	v.fields["z"] = &orderedValue{typ: jsonNumber, numStr: "1"}
	v.fields["a"] = &orderedValue{typ: jsonNumber, numStr: "2"}
	v.fields["m"] = &orderedValue{typ: jsonNumber, numStr: "3"}
	v.fields["b"] = &orderedValue{typ: jsonNumber, numStr: "4"}

	b, err := v.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"z":1,"a":2,"m":3,"b":4}`
	if string(b) != expected {
		t.Fatalf("expected %q, got %q", expected, string(b))
	}
}

func TestMinifyExpandRoundTrip(t *testing.T) {
	original := []byte(`{"log":{"version":"1.2","creator":{"name":"Test"}}}`)

	minified, err := MinifyJSON(original)
	if err != nil {
		t.Fatal(err)
	}

	expanded, err := ExpandJSON(minified)
	if err != nil {
		t.Fatal(err)
	}

	// Expand should add indentation
	if !strings.Contains(string(expanded), "\n") {
		t.Fatal("ExpandJSON should produce indented output")
	}

	// Minify should remove whitespace
	if strings.Contains(string(minified), "\n") {
		t.Fatal("MinifyJSON should produce single-line output")
	}

	// Re-minify should match
	reMinified, err := MinifyJSON(expanded)
	if err != nil {
		t.Fatal(err)
	}
	if string(reMinified) != string(minified) {
		t.Fatalf("round-trip failed:\noriginal:  %q\nre-minify: %q", string(minified), string(reMinified))
	}
}

func TestNumberPreservation(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"integer", `{"n":42}`},
		{"float", `{"n":3.14}`},
		{"scientific", `{"n":1.5e10}`},
		{"leading dot", `{"n":0.5}`},
		{"trailing zero", `{"n":147.0}`},
		{"negative", `{"n":-10}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := decodeOrdered([]byte(tc.json))
			if err != nil {
				t.Fatal(err)
			}
			b, err := v.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tc.json {
				t.Fatalf("expected %q, got %q", tc.json, string(b))
			}
		})
	}
}

func TestJSONQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", `"hello"`},
		{`say "hi"`, `"say \"hi\""`},
		{"a\nb", `"a\nb"`},
		{"a\\b", `"a\\b"`},
		{"tab\there", `"tab\there"`},
	}

	for _, tc := range tests {
		got := string(jsonQuote(tc.input))
		if got != tc.expected {
			t.Fatalf("jsonQuote(%q): expected %q, got %q", tc.input, tc.expected, got)
		}
	}
}

func TestArrayOrdered(t *testing.T) {
	data := []byte(`[3,1,2]`)
	v, err := decodeOrdered(data)
	if err != nil {
		t.Fatal(err)
	}
	if v.typ != jsonArray {
		t.Fatal("expected array")
	}
	if len(v.items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(v.items))
	}
	if v.items[0].numStr != "3" || v.items[1].numStr != "1" || v.items[2].numStr != "2" {
		t.Fatalf("unexpected array contents: %+v", v.items)
	}
}

func TestNestedOrdered(t *testing.T) {
	data := []byte(`{"outer":{"inner":"val"},"list":[1,2]}`)
	v, err := decodeOrdered(data)
	if err != nil {
		t.Fatal(err)
	}
	if v.typ != jsonObject {
		t.Fatal("expected object")
	}
	if len(v.keys) != 2 || v.keys[0] != "outer" || v.keys[1] != "list" {
		t.Fatalf("unexpected keys: %v", v.keys)
	}
}
