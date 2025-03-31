package ast

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Underline = "\033[4m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Purple    = "\033[35m"
	Cyan      = "\033[36m"
	White     = "\033[37m"
)

func NodeTypeName(node Node) string {
	return reflect.TypeOf(node).Elem().Name()
}

func PrettyPrint(node Node) string {
	return prettyPrintNode(node, 0)
}

func prettyPrintNode(node Node, level int) string {
	if node == nil {
		return Blue + "nil" + Reset
	}

	indent := strings.Repeat("  ", level)
	typeName := NodeTypeName(node)

	// Different colors for different node types
	var typeColor string
	switch {
	case strings.HasSuffix(typeName, "Statement"):
		typeColor = Yellow // Statements are yellow
	case strings.HasSuffix(typeName, "Expression"):
		typeColor = Green // Expressions are green
	case strings.HasSuffix(typeName, "Literal"):
		typeColor = Cyan // Literals are cyan
	default:
		typeColor = Purple // Other nodes are purple
	}

	result := fmt.Sprintf("%s%s%s%s%s: ", indent, Bold, typeColor, typeName, Reset)

	switch n := node.(type) {
	case *Program:
		result += "{\n"
		for _, stmt := range n.Statements {
			result += prettyPrintNode(stmt, level+1) + "\n"
		}
		result += indent + "}"

	case *LetStatement:
		result += fmt.Sprintf("let %s = ", n.Name.Value)
		if n.Value != nil {
			result += "\n" + prettyPrintNode(n.Value, level+1)
		} else {
			result += Blue + "nil" + Reset
		}

	case *ReturnStatement:
		result += "return "
		if n.ReturnValue != nil {
			result += "\n" + prettyPrintNode(n.ReturnValue, level+1)
		} else {
			result += Blue + "nil" + Reset
		}

	case *ExpressionStatement:
		if n.Expression != nil {
			result += "\n" + prettyPrintNode(n.Expression, level+1)
		} else {
			result += Blue + "nil" + Reset
		}

	case *BlockStatement:
		result += "{\n"
		for _, stmt := range n.Statements {
			result += prettyPrintNode(stmt, level+1) + "\n"
		}
		result += indent + "}"

	case *IfExpression:
		result += "if "
		result += prettyPrintNodeWithoutParens(n.Condition, level) + " "
		result += prettyPrintNode(n.Consequence, level)
		if n.Alternative != nil {
			result += " else " + prettyPrintNode(n.Alternative, level)
		}

	case *FunctionLiteral:
		result += "fn"
		if n.Name != nil {
			result += " " + n.Name.Value
		}
		result += "("
		for i, param := range n.Parameters {
			result += prettyPrintNode(param, 0)
			if i < len(n.Parameters)-1 {
				result += ", "
			}
		}
		result += ") " + prettyPrintNode(n.Body, level)

	case *CallExpression:
		result += prettyPrintNode(n.Function, level) + "("
		for i, arg := range n.Arguments {
			result += prettyPrintNode(arg, 0)
			if i < len(n.Arguments)-1 {
				result += ", "
			}
		}
		result += ")"

	case *PrefixExpression:
		result += fmt.Sprintf("%s%s", n.Operator, prettyPrintNode(n.Right, level))

	case *PostfixExpression:
		result += fmt.Sprintf("%s%s", prettyPrintNode(n.Left, level), n.Operator)

	case *InfixExpression:
		result += "("
		result += prettyPrintNode(n.Left, 0)
		result += " " + n.Operator + " "
		result += prettyPrintNode(n.Right, 0)
		result += ")"

	case *IntegerLiteral:
		result += fmt.Sprintf("%s%d%s", Red, n.Value, Reset)

	case *FloatLiteral:
		result += fmt.Sprintf("%s%f%s", Red, n.Value, Reset)

	case *StringLiteral:
		result += fmt.Sprintf("%s\"%s\"%s", Red, n.Value, Reset)

	case *Boolean:
		result += fmt.Sprintf("%s%v%s", Red, n.Value, Reset)

	case *Identifier:
		result += fmt.Sprintf("%s%s%s", Red, n.Value, Reset)

	case *ArrayLiteral:
		result += "["
		for i, elem := range n.Elements {
			result += prettyPrintNode(elem, 0)
			if i < len(n.Elements)-1 {
				result += ", "
			}
		}
		result += "]"

	case *HashLiteral:
		result += "{"
		i := 0
		for _, pair := range n.Pairs {
			result += prettyPrintNode(pair.Key, 0) + ": " + prettyPrintNode(pair.Value, 0)
			i++
			if i < len(n.Pairs) {
				result += ", "
			}
		}
		result += "}"

	case *IndexExpression:
		result += prettyPrintNode(n.Left, level) + "[" + prettyPrintNode(n.Index, 0) + "]"

	case *DotExpression:
		result += prettyPrintNode(n.Left, level) + "." + n.Property.Value

	case *MethodCallExpression:
		result += prettyPrintNode(n.Object, level) + "." + n.Method.Value + "("
		for i, arg := range n.Arguments {
			result += prettyPrintNode(arg, 0)
			if i < len(n.Arguments)-1 {
				result += ", "
			}
		}
		result += ")"

	case *AssignmentExpression:
		result += prettyPrintNode(n.Left, level) + " = " + prettyPrintNode(n.Value, level)

	case *RepeatStatement:
		result += "repeat " + n.Iterator.Value + " in " + prettyPrintNode(n.Collection, 0) + " " + prettyPrintNode(n.Body, level)

	case *LoadStatement:
		result += "load " + prettyPrintNode(n.Path, 0)
		if n.Alias != nil {
			result += " as " + n.Alias.Value
		}

	case *PrintStatement:
		result += "print("
		for i, arg := range n.Arguments {
			if i > 0 {
				result += ", "
			}
			result += prettyPrintNode(arg, 0)
		}
		result += ")"

	default:
		result += fmt.Sprintf("%s (using default String method)", node.String())
	}

	return result
}

func prettyPrintNodeWithoutParens(node Node, level int) string {
	if _, isInfix := node.(*InfixExpression); isInfix {
		n := node.(*InfixExpression)
		return fmt.Sprintf("%s %s %s",
			prettyPrintNode(n.Left, 0),
			n.Operator,
			prettyPrintNode(n.Right, 0))
	}
	return prettyPrintNode(node, level)
}

// NoColorPrettyPrint provides a version without color codes for environments that don't support them
func NoColorPrettyPrint(node Node) string {
	s := PrettyPrint(node)
	// Strip all color codes
	s = strings.ReplaceAll(s, Reset, "")
	s = strings.ReplaceAll(s, Bold, "")
	s = strings.ReplaceAll(s, Underline, "")
	s = strings.ReplaceAll(s, Red, "")
	s = strings.ReplaceAll(s, Green, "")
	s = strings.ReplaceAll(s, Yellow, "")
	s = strings.ReplaceAll(s, Blue, "")
	s = strings.ReplaceAll(s, Purple, "")
	s = strings.ReplaceAll(s, Cyan, "")
	s = strings.ReplaceAll(s, White, "")
	return s
}

func PrettyPrintToTerminal(node Node) {
	fmt.Println(PrettyPrint(node))
}
