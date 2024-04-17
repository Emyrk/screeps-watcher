package callgrind

import (
	"time"
)

type Node struct {
	Function *Function
	Cost     time.Duration

	Children map[string]*Edge
}

type Edge struct {
	Node
}

func (p *Profile) Callstack() error {

	return nil
}

//func (p *Profile) CallTree() error {
//	roots := p.Roots()
//	if len(roots) != 1 {
//		return fmt.Errorf("expected 1 root, got %d", len(roots))
//	}
//
//	root := &Node{
//		Function: roots[0],
//		Cost:     roots[0].Cost,
//	}
//
//	return nil
//}
//
//func (p *Profile) recurseTree(n *Node) {
//	for _, call := range n.Function.calls {
//		child, ok := n.Children[call.CalleeId]
//		if ok {
//			// This branch is over
//			return
//		}
//
//		callee, _ := p.GetFunction(call.CalleeId)
//		node := &Node{
//			Function: callee,
//			Cost:     callee.Cost,
//		}
//		n.Children = append(n.Children, node)
//		p.recurseTree(node)
//	}
//}
