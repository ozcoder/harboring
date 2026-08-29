package har

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func ShrinkContent(root *Node) {
	shrinkWalk(root, false)
}

func shrinkWalk(n *Node, inContent bool) {
	if n.Key == "content" {
		inContent = true
	}

	if n.Kind == String && n.Value != nil {
		if s, ok := n.Value.(string); ok {
			trimmed := strings.TrimSpace(s)
			if !inContent || n.Key == "mimeType" || n.Key == "encoding" || n.Key == "comment" {
				n.Value = trimmed
			}
		}
	}

	for _, child := range n.Children {
		shrinkWalk(child, inContent)
	}
}

func ShrinkFile(inputPath, outputPath string) error {
	data, err := ReadFileLimited(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	root, err := ParseBytes(data)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	ShrinkContent(root)

	output, err := ExportJSON(root)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func MinifyJSON(data []byte) ([]byte, error) {
	dec, err := Decompress(data)
	if err != nil {
		return nil, err
	}
	raw, err := decodeOrdered(dec)
	if err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return raw.MarshalJSON()
}

func ExpandJSON(data []byte) ([]byte, error) {
	dec, err := Decompress(data)
	if err != nil {
		return nil, err
	}
	raw, err := decodeOrdered(dec)
	if err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	minified, err := raw.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, minified, "", "  "); err != nil {
		return nil, fmt.Errorf("indent json: %w", err)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func MinifyFile(inputPath, outputPath string) error {
	data, err := ReadFileLimited(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	output, err := MinifyJSON(data)
	if err != nil {
		return fmt.Errorf("minify: %w", err)
	}

	if err := WriteFileAuto(outputPath, output); err != nil {
		return err
	}

	return nil
}

func ExpandFile(inputPath, outputPath string) error {
	data, err := ReadFileLimited(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	output, err := ExpandJSON(data)
	if err != nil {
		return fmt.Errorf("expand: %w", err)
	}

	if err := WriteFileAuto(outputPath, output); err != nil {
		return err
	}

	return nil
}
