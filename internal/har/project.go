package har

import (
	"path"
	"strings"
)

// ProjectFilters returns a new root Node containing only the fields specified
// by dot-path expressions. Paths are relative to each entry (e.g.
// "timings.wait" keeps entry.timings.wait). A path that matches an entire
// object key (e.g. "timings") keeps the whole subtree. Log metadata and pages
// pass through unfiltered; only entry fields are projected.
func ProjectFilters(root *Node, paths []string) *Node {
	tree := buildPathTree(paths)
	if len(tree) == 0 {
		return root
	}
	if root == nil || root.Kind != Object {
		return root
	}

	logNode := findChild(root, "log")
	if logNode == nil {
		return root
	}

	outLog := &Node{Key: "log", Kind: Object, Depth: 1}
	for _, child := range logNode.Children {
		if child.Key == "entries" {
			outEntries := &Node{Key: "entries", Kind: Array, Depth: 2, Parent: outLog}
			for _, entry := range child.Children {
				outEntries.Children = append(outEntries.Children, projectNode(entry, tree))
			}
			outLog.Children = append(outLog.Children, outEntries)
		} else {
			outLog.Children = append(outLog.Children, child)
		}
	}

	return &Node{Kind: Object, Children: []*Node{outLog}}
}

// buildPathTree converts a list of dot-separated paths into a nested map.
// A nil value means "keep entire subtree"; a non-nil map means recurse deeper.
func buildPathTree(paths []string) map[string]interface{} {
	tree := make(map[string]interface{})
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		parts := strings.Split(p, ".")
		current := tree
		for i, part := range parts {
			existing, ok := current[part]
			if !ok {
				current[part] = interface{}(nil)
				if i < len(parts)-1 {
					next := make(map[string]interface{})
					current[part] = next
					current = next
				}
			} else if i < len(parts)-1 {
				if m, ok := existing.(map[string]interface{}); ok {
					current = m
				} else {
					break
				}
			}
		}
	}
	return tree
}

// projectNode returns a new node matching n's kind and key, with only the
// children whose keys match a pattern in paths. Glob patterns (*, ?, [...])
// are supported; nested paths are applied recursively.
func projectNode(n *Node, paths map[string]interface{}) *Node {
	if n == nil {
		return nil
	}
	out := cloneShallow(n)
	switch n.Kind {
	case Object:
		for _, child := range n.Children {
			val, ok := matchPath(paths, child.Key)
			if !ok {
				continue
			}
			if val == nil {
				out.Children = append(out.Children, child)
			} else {
				projected := projectNode(child, val.(map[string]interface{}))
				if projected != nil {
					out.Children = append(out.Children, projected)
				}
			}
		}
	case Array:
		if len(paths) > 0 {
			for _, child := range n.Children {
				projected := projectNode(child, paths)
				if projected != nil {
					out.Children = append(out.Children, projected)
				}
			}
		}
	}
	return out
}

// matchPath looks up key in paths, first by exact match, then by trying each
// glob pattern. Returns the value and true if a match was found.
func matchPath(paths map[string]interface{}, key string) (interface{}, bool) {
	if val, ok := paths[key]; ok {
		return val, true
	}
	for pattern, val := range paths {
		if strings.ContainsAny(pattern, "*?[") {
			if matched, _ := path.Match(pattern, key); matched {
				return val, true
			}
		}
	}
	return nil, false
}

// cloneShallow creates a copy of n with the same scalar fields but no children.
func cloneShallow(n *Node) *Node {
	return &Node{
		Key:       n.Key,
		Value:     n.Value,
		NumStr:    n.NumStr,
		Kind:      n.Kind,
		Parent:    n.Parent,
		IsKnown:   n.IsKnown,
		Depth:     n.Depth,
		Collapsed: n.Collapsed,
	}
}
