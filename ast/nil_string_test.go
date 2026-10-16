package ast

import "testing"

// Nodes with the parts missing that code which failed to parse leaves out
// can still be shown, as patterns and errors do
func TestStringWithMissingParts(t *testing.T) {
	a := &Identifier{Value: "a"}
	nodes := map[Node]string{
		&AssignmentExpression{Left: a}:           "a = ",
		&AssignmentExpression{Value: a}:          " = a",
		&InfixExpression{Left: a, Operator: "+"}: "(a + )",
		&PrefixExpression{Operator: "-"}:         "(-)",
		&AwaitExpression{}:                       "await ",
		&SpreadExpression{}:                      "...",
	}
	for n, want := range nodes {
		if got := n.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
