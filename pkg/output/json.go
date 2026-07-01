package output

import (
	"encoding/json"
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/neilotoole/jsoncolor"
)

// IsTTY can be overridden in tests to simulate terminal output.
var IsTTY = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && isatty.IsTerminal(f.Fd())
}

func PrintJSON(w io.Writer, v any) error {
	if IsTTY(w) {
		enc := jsoncolor.NewEncoder(w)
		enc.SetIndent("", "  ")
		clrs := defaultColors()
		enc.SetColors(&clrs)
		return enc.Encode(v)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func defaultColors() jsoncolor.Colors {
	return jsoncolor.Colors{
		Null:   jsoncolor.Color("\033[2m"),
		Bool:   jsoncolor.Color("\033[33m"),
		Number: jsoncolor.Color("\033[36m"),
		String: jsoncolor.Color("\033[32m"),
		Key:    jsoncolor.Color("\033[1;34m"),
		Punc:   jsoncolor.Color("\033[2m"),
	}
}
