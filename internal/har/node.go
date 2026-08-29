package har

import "strconv"

type NodeKind int

const (
	Object NodeKind = iota
	Array
	String
	Number
	Bool
	Null
)

type Node struct {
	Key       string
	Value     interface{}
	NumStr    string // original JSON number literal, preserved verbatim
	Kind      NodeKind
	Children  []*Node
	Collapsed bool
	Parent    *Node
	IsKnown   bool
	Depth     int
}

func (n *Node) Leaf() bool {
	return len(n.Children) == 0
}

// NumberText returns the original JSON number literal for Number nodes. Nodes
// built from the ordered decoder carry the verbatim literal (NumStr); nodes
// constructed in code fall back to formatting the float64 value.
func (n *Node) NumberText() string {
	if n.NumStr != "" {
		return n.NumStr
	}
	return strconv.FormatFloat(n.Value.(float64), 'f', -1, 64)
}

func (n *Node) ToggleCollapse() {
	if n.Kind != Object && n.Kind != Array {
		return
	}
	n.Collapsed = !n.Collapsed
}

func (n *Node) ExpandAll() {
	if n.Kind == Object || n.Kind == Array {
		n.Collapsed = false
	}
	for _, child := range n.Children {
		child.ExpandAll()
	}
}

func (n *Node) CollapseAll() {
	if n.Kind == Object || n.Kind == Array {
		n.Collapsed = true
	}
	for _, child := range n.Children {
		child.CollapseAll()
	}
}

func (n *Node) FindNode(index int) *Node {
	counter := 0
	return findNodeWalk(n, &counter, index)
}

func findNodeWalk(n *Node, counter *int, target int) *Node {
	if *counter == target {
		return n
	}
	*counter++
	if n.Leaf() || n.Collapsed {
		return nil
	}
	for _, child := range n.Children {
		result := findNodeWalk(child, counter, target)
		if result != nil {
			return result
		}
	}
	return nil
}
