package evaluator

import (
	"strings"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/order"
	"github.com/migueltarga/TargaScript/strfmt"
)

// Methods of each type, used to suggest a name when a method doesn't
// exist. The methods in moreStringMethods and moreHashMethods are added
// to these.
var (
	stringMethods = []string{"trim", "lower", "upper", "replace", "split", "contains", "startsWith", "endsWith", "indexOf"}
	arrayMethods  = []string{"first", "last", "rest", "insert", "join", "map", "filter", "reduce", "contains", "indexOf", "find", "slice", "sort", "reverse",
		"count", "every", "flatten", "groupBy", "max", "min", "some", "sortBy", "sum", "unique", "pop", "remove", "removeAt", "shift", "unshift"}
	hashMethods = []string{"keys", "values", "has", "length"}
)

// The names other languages use, with what to write instead
var (
	otherLanguageNames = [][2]string{
		{"len", "use .length, as in list.length"},
		{"int", "use toInt(value)"},
		{"parseInt", "use toInt(value)"},
		{"float", "use toFloat(value)"},
		{"parseFloat", "use toFloat(value)"},
		{"str", "use toString(value)"},
		{"String", "use toString(value)"},
		{"console", "use print(...)"},
		{"println", "use print(...)"},
		{"puts", "use print(...)"},
		{"echo", "use print(...)"},
		{"typeof", "use type(value)"},
		{"require", `use load("file.tg")`},
		{"include", `use load("file.tg")`},
		{"nil", "use null"},
		{"None", "use null"},
		{"undefined", "use null"},
		{"True", "use true"},
		{"False", "use false"},
		{"Math", "math functions are builtins: abs, sqrt, pow, round, floor, ceil, min, max, random"},
		{"JSON", "use parseJSON(text) and toJSON(value)"},
	}
	otherLanguageStringMethods = [][2]string{
		{"length", "length is a property: write .length, without ()"},
		{"toUpperCase", "use upper()"},
		{"toUpper", "use upper()"},
		{"toLowerCase", "use lower()"},
		{"toLower", "use lower()"},
		{"strip", "use trim()"},
		{"includes", "use contains(text) or the in operator"},
		{"substring", "use slice(start, end)"},
		{"substr", "use slice(start, end)"},
		{"charAt", "use text[index]"},
		{"size", "use .length"},
		{"len", "use .length"},
	}
	otherLanguageArrayMethods = [][2]string{
		{"length", "length is a property: write .length, without ()"},
		{"push", "use insert(value)"},
		{"append", "use insert(value)"},
		{"add", "use insert(value)"},
		{"includes", "use contains(value) or the in operator"},
		{"forEach", "use repeat item in list { ... }"},
		{"each", "use repeat item in list { ... }"},
		{"flat", "use flatten()"},
		{"concat", "use list + other"},
		{"any", "use some(fn)"},
		{"all", "use every(fn)"},
		{"distinct", "use unique()"},
		{"uniq", "use unique()"},
		{"size", "use .length"},
		{"len", "use .length"},
	}
	otherLanguageHashMethods = [][2]string{
		{"hasOwnProperty", `use has(key) or "key" in object`},
		{"containsKey", `use has(key) or "key" in object`},
		{"hasKey", `use has(key) or "key" in object`},
		{"items", "use entries()"},
		{"size", "use .length"},
		{"len", "use .length"},
	}
)

func init() {
	for name := range moreStringMethods {
		stringMethods = append(stringMethods, name)
	}
	for name := range moreHashMethods {
		hashMethods = append(hashMethods, name)
	}
}

// hint returns what to write instead of name when another language uses
// it, else a similar name from candidates.
func hint(name string, otherLanguages [][2]string, candidates []string) string {
	if instead, ok := lookup(otherLanguages, name); ok {
		return ". In TargaScript, " + instead
	}
	return suggestion(name, candidates)
}

// lookup finds the value of a key in a list of pairs. The lists of hints
// are short, and a list takes less room than a map in the WebAssembly build.
func lookup(pairs [][2]string, key string) (string, bool) {
	for _, pair := range pairs {
		if pair[0] == key {
			return pair[1], true
		}
	}
	return "", false
}

// typeName describes the type of a value for error messages, with an
// article: "an integer", "a string".
func typeName(obj object.Object) string {
	switch obj.Type() {
	case object.INTEGER_OBJ:
		return "an integer"
	case object.FLOAT_OBJ:
		return "a float"
	case object.STRING_OBJ:
		return "a string"
	case object.BOOLEAN_OBJ:
		return "a boolean"
	case object.NULL_OBJ:
		return "null"
	case object.ARRAY_OBJ:
		return "an array"
	case object.HASH_OBJ:
		if typ := obj.(*object.Hash).Struct(); typ != nil {
			return article(typ.Name)
		}
		return "an object"
	case object.STRUCT_OBJ:
		return "a struct"
	case object.ENUM_OBJ:
		return "an enum"
	case object.ENUM_VALUE_OBJ:
		return article(obj.(*object.EnumValue).Enum.Name)
	case object.FUNCTION_OBJ, object.BUILTIN_OBJ:
		return "a function"
	}
	if _, ok := obj.(object.External); ok {
		return "an external value"
	}
	return strings.ToLower(string(obj.Type()))
}

// kindName is the name of the type of a value, as type() returns it:
// "integer", "string", "object", "function"...
func kindName(obj object.Object) string {
	switch obj.Type() {
	case object.HASH_OBJ:
		if typ := obj.(*object.Hash).Struct(); typ != nil {
			return typ.Name
		}
		return "object"
	case object.ENUM_VALUE_OBJ:
		return obj.(*object.EnumValue).Enum.Name
	case object.FUNCTION_OBJ, object.BUILTIN_OBJ:
		return "function"
	}
	return strings.ToLower(string(obj.Type()))
}

// operatorError reports an operator that can't be used with its operands,
// with what to write instead for the usual mix-ups.
func operatorError(line, column int, operator string, left, right object.Object) *object.Error {
	return newError(line, column, "cannot use %q with %s and %s%s", operator, typeName(left), typeName(right), operatorHint(operator, left, right))
}

func operatorHint(operator string, left, right object.Object) string {
	isNumber := func(o object.Object) bool { return o.Type() == object.INTEGER_OBJ || o.Type() == object.FLOAT_OBJ }
	switch {
	case left.Type() == object.NULL_OBJ || right.Type() == object.NULL_OBJ:
		return ". One side is null: check where it comes from, or give it a default with ??, as in (x ?? 0)"
	case operator == "+" && isNumber(left) && right.Type() == object.STRING_OBJ:
		return `. To make text, put the number in the string: "${n} items", or start with the string`
	case operator == "+" && left.Type() == object.ARRAY_OBJ:
		return ". To add an element, use list.insert(x), or list + [x] for a new array"
	case operator == "+" && left.Type() == object.HASH_OBJ && right.Type() == object.HASH_OBJ:
		return ". To combine objects, use {...a, ...b}"
	}
	return ""
}

// NotDefined reports an unknown name, suggesting a similar one from
// candidates, the builtins or print.
func NotDefined(line, column int, name string, candidates []string) *object.Error {
	candidates = append([]string{}, candidates...)
	for builtin := range builtins {
		candidates = append(candidates, builtin)
	}
	return newError(line, column, "%q is not defined%s", name, hint(name, otherLanguageNames, candidates))
}

// suggestion returns ". Did you mean "x"?" for the candidate closest to
// name, or "" when none is close enough to be a likely typo.
func suggestion(name string, candidates []string) string {
	// Allow one typo in names of three to six letters, and two in longer
	// ones. Shorter names only match when they differ in case.
	limit := min(2, (len(name)+1)/4)
	best, bestDistance := "", limit+1

	order.Strings(candidates)
	for _, candidate := range candidates {
		if candidate == name {
			continue
		}
		// Differences in case alone count as the closest match
		distance := editDistance(strings.ToLower(name), strings.ToLower(candidate))
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}

	if best == "" {
		return ""
	}
	return strfmt.Sprintf(". Did you mean %q?", best)
}

// editDistance is the number of letters to insert, delete, replace or swap
// with their neighbor to turn a into b (optimal string alignment distance).
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
}

// article puts a or an before a name, by its first letter.
func article(name string) string {
	if name != "" && strings.ContainsRune("AEIOUaeiou", rune(name[0])) {
		return "an " + name
	}
	return "a " + name
}
