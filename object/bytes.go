package object

import "strconv"

// BYTES_OBJ is the type of binary data: bytes from 0 to 255.
const BYTES_OBJ = "BYTES"

// Bytes is binary data, like the contents of a file or a Uint8Array.
type Bytes struct {
	Value []byte
}

// bytesShown is how many bytes Inspect shows before saying how many more
// there are.
const bytesShown = 32

func (b *Bytes) Type() ObjectType { return BYTES_OBJ }

// Inspect shows the bytes in hexadecimal: bytes(48 69).
func (b *Bytes) Inspect() string {
	shown := b.Value[:min(len(b.Value), bytesShown)]
	text := "bytes(" + Hex(shown, true)
	if len(shown) < len(b.Value) {
		text += " ... " + strconv.Itoa(len(b.Value)-len(shown)) + " more"
	}
	return text + ")"
}

// Hex writes data in hexadecimal, with spaces between the bytes or not.
func Hex(data []byte, spaced bool) string {
	out := make([]byte, 0, len(data)*3)
	for i, c := range data {
		if spaced && i > 0 {
			out = append(out, ' ')
		}
		out = append(out, "0123456789abcdef"[c>>4], "0123456789abcdef"[c&15])
	}
	return string(out)
}
