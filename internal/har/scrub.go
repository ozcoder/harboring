package har

import (
	"fmt"
	"os"
)

func ScrubContent(root *Node) {
	scrubWalk(root)
}

func scrubWalk(n *Node) {
	if n.Key == "text" || n.Key == "encoding" {
		if n.Parent != nil && n.Parent.Key == "content" {
			n.Value = ""
			return
		}
	}
	for _, child := range n.Children {
		scrubWalk(child)
	}
}

func ScrubFile(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	root, err := ParseBytes(data)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	ScrubContent(root)

	output, err := ExportJSON(root)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}
