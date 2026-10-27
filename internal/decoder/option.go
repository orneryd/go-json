package decoder

import "context"

type OptionFlags uint8

const (
	FirstWinOption OptionFlags = 1 << iota
	ContextOption
	PathOption
	// UseNumberOption decodes a number into interface{} as json.Number instead of float64.
	UseNumberOption
	// DisallowUnknownFieldsOption makes an object key which matches no field of the struct an error.
	DisallowUnknownFieldsOption
	// NoCopyStringOption makes the decoded strings refer to the input instead of a copy of their bytes.
	NoCopyStringOption
	// PreserveInvalidUTF8Option keeps the invalid UTF-8 of a decoded string as it is in the input, instead of
	// replacing it with utf8.RuneError as encoding/json does ( see decodeLiteral ). The escapes are decoded, and a
	// lone escaped surrogate is still written as utf8.RuneError: only the bytes of the input are kept.
	PreserveInvalidUTF8Option
)

type Option struct {
	Flags   OptionFlags
	Context context.Context
	Path    *Path
}
