package report

import (
	"encoding/json"
	"io"
)

// WriteJSON writes the report as JSON to w.
func WriteJSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
