package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestBytes(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`bytes("Hi é")`, "bytes(48 69 20 c3 a9)"},
		{`let b = bytes("Hi")` + "\n[b.length, type(b), b[0], b[-1], b[2]]", `[2, "bytes", 72, 105, null]`},
		{"let b = bytes(2)\nb[0] = 104\nb[1] = 105\nb.text()", "hi"},
		{"bytes([0, 15, 255]).hex()", "000fff"},
		{"bytes([1, 2]).toArray()", "[1, 2]"},
		{"bytes([1, 2]) + bytes([3])", "bytes(01 02 03)"},
		{"[bytes([1]) == bytes([1]), bytes([1]) == bytes([2]), bytes([1]) == [1]]", "[true, false, false]"},
		{"let b = bytes(8)\nb.write(0, 4, 0x04034b50)\nb.writeBE(4, 2, 0xcafe)\nb", "bytes(50 4b 03 04 ca fe 00 00)"},
		{"let b = bytes([0x50, 0x4b, 3, 4])\n[b.read(0, 4), b.readBE(0, 2), b.read(2, 1)]", "[67324752, 20555, 3]"},
		{"let b = bytes(8)\nb.write(0, 8, -1)\nb.read(0, 8)", "-1"},
		{"bytes([1, 2, 3, 4]).slice(1, 3)", "bytes(02 03)"},
		{"bytes([1, 2, 3, 4]).slice(-1)", "bytes(04)"},
		{"let s = 0\nrepeat x in bytes([1, 2, 3]) { s += x }\ns", "6"},
		{"let b = bytes(300)\nrepeat i in 0..299 { b[i] = i % 256 }\nlet s = 0\nrepeat i in 0..299 { s += b[i] }\ns", "33586"},
		{"let a = bytes([1])\nlet b = bytes(a)\nb[0] = 2\na[0]", "1"},
		{"bytes(33)", "bytes(00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ... 1 more)"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestBytesErrors(t *testing.T) {
	tests := []struct{ input, message string }{
		{"bytes([256])", "numbers from 0 to 255, got 256 at 0"},
		{"bytes(-1)", "size from 0 to"},
		{"bytes(3)[0] = 300", "a byte is a number from 0 to 255, got 300"},
		{"bytes(3)[5] = 1", "index out of bounds: 5"},
		{"bytes(3).read(1, 4)", "outside the 3 bytes"},
		{"bytes(3).read(0, 3)", "a position and a size of 1, 2, 4 or 8"},
		{"bytes(3).write(0, 1, 256)", "256 doesn't fit in 1 byte(s)"},
		{"bytes(1).lenght", `Did you mean "length"?`},
	}
	for _, tt := range tests {
		result, ok := testEval(tt.input).(*object.Error)
		if !ok || !strings.Contains(result.Message, tt.message) {
			t.Errorf("%s: got %v, want %q", tt.input, result, tt.message)
		}
	}
}
