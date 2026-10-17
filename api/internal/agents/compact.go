package agents

import (
	"bytes"
	"encoding/json"
	"regexp"
	"slices"
)

// Compact tool results (EP-152), in the spirit of rtk, which compresses a
// command's output before a coding agent reads it: what a tool returns is
// rewritten so it costs fewer tokens and says the same. Nothing the model
// could act on is lost: ids, numbers, and booleans stay; only empty values,
// sub-minute time, and repeated field names go. A list of three or more
// records with the same fields becomes {"cols": […], "rows": [[…], …]}.

var timestamp = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}):\d{2}(?:\.\d+)?(Z|[+-]\d{2}:\d{2})$`)

// compactNote tells the model how compact results read.
const compactNote = "Tool results are compact: empty fields are left out, times are to the minute, and a list of records with the same fields comes as {\"cols\": [names], \"rows\": [[values in that order], …]}."

// compact rewrites a JSON value and returns it with the size it had.
func compact(b []byte) (out []byte, raw int) {
	var v any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		return b, len(b)
	}
	v, _ = shrink(v)
	out, err := json.Marshal(v)
	if err != nil {
		return b, len(b)
	}
	return out, len(b)
}

// shrink returns the compact value and whether anything is left of it.
func shrink(v any) (any, bool) {
	switch x := v.(type) {
	case nil:
		return nil, false
	case string:
		if x == "" {
			return nil, false
		}
		if m := timestamp.FindStringSubmatch(x); m != nil {
			return m[1] + m[2], true
		}
		return x, true
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			if c, ok := shrink(e); ok {
				out[k] = c
			}
		}
		return out, len(out) > 0
	case []any:
		if t, ok := table(x); ok {
			return t, true
		}
		out := make([]any, 0, len(x))
		for _, e := range x {
			if c, ok := shrink(e); ok {
				out = append(out, c)
			}
		}
		return out, len(out) > 0
	default:
		return v, true
	}
}

// table turns three or more records that share their fields into columns
// and rows. A column empty in every row is left out; an empty cell is null.
func table(list []any) (map[string]any, bool) {
	if len(list) < 3 {
		return nil, false
	}
	seen := map[string]int{}
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			return nil, false
		}
		for k := range m {
			seen[k]++
		}
	}
	// Worth it only when most records have most fields.
	shared := 0
	for _, n := range seen {
		if n*4 >= len(list)*3 {
			shared++
		}
	}
	if shared*4 < len(seen)*3 {
		return nil, false
	}
	cells := make([]map[string]any, len(list))
	used := map[string]bool{}
	for i, e := range list {
		cells[i] = map[string]any{}
		for k, v := range e.(map[string]any) {
			if c, ok := shrink(v); ok {
				cells[i][k], used[k] = c, true
			}
		}
	}
	cols := make([]string, 0, len(used))
	for k := range used {
		cols = append(cols, k)
	}
	slices.Sort(cols)
	rows := make([][]any, len(list))
	for i := range list {
		row := make([]any, len(cols))
		for j, k := range cols {
			row[j] = cells[i][k]
		}
		rows[i] = row
	}
	return map[string]any{"cols": cols, "rows": rows}, true
}

// must is a marshalled value; tool results are plain data that always marshal.
func must(b []byte, err error) []byte {
	if err != nil {
		return []byte("null")
	}
	return b
}
