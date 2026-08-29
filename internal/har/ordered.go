package har

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type jsonType int

const (
	jsonNull jsonType = iota
	jsonBool
	jsonNumber
	jsonString
	jsonArray
	jsonObject
)

// maxParseDepth caps the recursion of the JSON decoder. Real HAR files nest
// only a handful of levels deep; an unbounded depth lets a malicious file
// trigger a goroutine stack overflow (panic / crash).
const maxParseDepth = 10000

type orderedValue struct {
	typ     jsonType
	boolVal bool
	numStr  string
	strVal  string
	items   []*orderedValue
	keys    []string
	fields  map[string]*orderedValue
}

func decodeOrdered(data []byte) (*orderedValue, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeOrderedFromDecoder(dec, 0)
	if err != nil {
		return nil, err
	}
	// consume any trailing data
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected trailing data")
	}
	return v, nil
}

func decodeOrderedFromDecoder(dec *json.Decoder, depth int) (*orderedValue, error) {
	if depth > maxParseDepth {
		return nil, fmt.Errorf("input nested too deeply (max depth %d)", maxParseDepth)
	}
	t, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("read token: %w", err)
	}

	switch delim := t.(type) {
	case json.Delim:
		switch delim {
		case '{':
			v := &orderedValue{typ: jsonObject, fields: make(map[string]*orderedValue)}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("read object key: %w", err)
				}
				key, ok := kt.(string)
				if !ok {
					return nil, fmt.Errorf("expected string key, got %T", kt)
				}
				child, err := decodeOrderedFromDecoder(dec, depth+1)
				if err != nil {
					return nil, fmt.Errorf("read object value for key %q: %w", key, err)
				}
				v.keys = append(v.keys, key)
				v.fields[key] = child
			}
			// consume '}'
			if _, err := dec.Token(); err != nil {
				return nil, fmt.Errorf("read closing brace: %w", err)
			}
			return v, nil

		case '[':
			v := &orderedValue{typ: jsonArray}
			for dec.More() {
				child, err := decodeOrderedFromDecoder(dec, depth+1)
				if err != nil {
					return nil, fmt.Errorf("read array element: %w", err)
				}
				v.items = append(v.items, child)
			}
			// consume ']'
			if _, err := dec.Token(); err != nil {
				return nil, fmt.Errorf("read closing bracket: %w", err)
			}
			return v, nil

		default:
			return nil, fmt.Errorf("unexpected delimiter: %v", delim)
		}

	case string:
		return &orderedValue{typ: jsonString, strVal: delim}, nil

	case json.Number:
		return &orderedValue{typ: jsonNumber, numStr: string(delim)}, nil

	case bool:
		return &orderedValue{typ: jsonBool, boolVal: delim}, nil

	case nil:
		return &orderedValue{typ: jsonNull}, nil
	}

	return nil, fmt.Errorf("unexpected token type: %T", t)
}

func jsonQuote(s string) []byte {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case '\b':
			buf.WriteString(`\u0008`)
		case '\f':
			buf.WriteString(`\u000c`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&buf, "\\u%04x", r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return buf.Bytes()
}

func (v *orderedValue) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	switch v.typ {
	case jsonNull:
		buf.WriteString("null")

	case jsonBool:
		if v.boolVal {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}

	case jsonNumber:
		buf.WriteString(v.numStr)

	case jsonString:
		buf.Write(jsonQuote(v.strVal))

	case jsonArray:
		buf.WriteByte('[')
		for i, item := range v.items {
			if i > 0 {
				buf.WriteByte(',')
			}
			b, err := item.MarshalJSON()
			if err != nil {
				return nil, err
			}
			buf.Write(b)
		}
		buf.WriteByte(']')

	case jsonObject:
		buf.WriteByte('{')
		for i, key := range v.keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.Write(jsonQuote(key))
			buf.WriteByte(':')
			b, err := v.fields[key].MarshalJSON()
			if err != nil {
				return nil, err
			}
			buf.Write(b)
		}
		buf.WriteByte('}')
	}

	return buf.Bytes(), nil
}

func nodeToOrdered(n *Node) *orderedValue {
	switch n.Kind {
	case Object:
		v := &orderedValue{typ: jsonObject, fields: make(map[string]*orderedValue)}
		for _, child := range n.Children {
			v.keys = append(v.keys, child.Key)
			v.fields[child.Key] = nodeToOrdered(child)
		}
		return v

	case Array:
		v := &orderedValue{typ: jsonArray}
		for _, child := range n.Children {
			v.items = append(v.items, nodeToOrdered(child))
		}
		return v

	case String:
		return &orderedValue{typ: jsonString, strVal: n.Value.(string)}

	case Number:
		return &orderedValue{typ: jsonNumber, numStr: n.NumberText()}

	case Bool:
		return &orderedValue{typ: jsonBool, boolVal: n.Value.(bool)}

	case Null:
		return &orderedValue{typ: jsonNull}
	}

	return &orderedValue{typ: jsonNull}
}
