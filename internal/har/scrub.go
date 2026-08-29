package har

import (
	"fmt"
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
	// Empty out cookies arrays (in request/response) while keeping the array node
	if n.Key == "cookies" && n.Kind == Array {
		n.Children = nil
	}
	for _, child := range n.Children {
		scrubWalk(child)
	}
}

func ScrubFile(inputPath, outputPath string) error {
	data, err := ReadFileLimited(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	dec, err := Decompress(data)
	if err != nil {
		return err
	}

	root, err := ParseBytes(dec)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	ScrubContent(root)

	output, err := ExportJSON(root)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	if err := WriteFileAuto(outputPath, output); err != nil {
		return err
	}

	return nil
}
