package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const defaultLimit = 30

type listOptions struct {
	limit      int
	skip       int
	sortBy     string
	order      string
	selectKeys []string
}

func parseListOptions(r *http.Request) listOptions {
	query := r.URL.Query()
	opts := listOptions{
		limit: defaultLimit,
		order: "asc",
	}

	if raw := query.Get("limit"); raw != "" {
		if number, err := strconv.Atoi(raw); err == nil && number >= 0 {
			opts.limit = number
		}
	}

	if raw := query.Get("skip"); raw != "" {
		if number, err := strconv.Atoi(raw); err == nil && number >= 0 {
			opts.skip = number
		}
	}

	opts.sortBy = query.Get("sortBy")

	if query.Get("order") == "desc" {
		opts.order = "desc"
	}

	if raw := query.Get("select"); raw != "" {
		opts.selectKeys = strings.Split(raw, ",")
	}

	return opts
}

func applyList(rows []map[string]any, opts listOptions) []map[string]any {
	out := rows

	if opts.sortBy != "" {
		out = sortRows(out, opts.sortBy, opts.order)
	}

	if opts.skip > 0 {
		if opts.skip >= len(out) {
			out = out[:0]
		} else {
			out = out[opts.skip:]
		}
	}

	if opts.limit > 0 && opts.limit < len(out) {
		out = out[:opts.limit]
	}

	if len(opts.selectKeys) > 0 {
		out = selectKeys(out, opts.selectKeys)
	}

	return out
}

func sortRows(rows []map[string]any, key, order string) []map[string]any {
	out := make([]map[string]any, len(rows))
	copy(out, rows)

	sort.SliceStable(out, func(i, j int) bool {
		comparison := compareValues(out[i][key], out[j][key])
		if order == "desc" {
			return comparison > 0
		}
		return comparison < 0
	})

	return out
}

func compareValues(left, right any) int {
	switch leftValue := left.(type) {
	case float64:
		if rightValue, ok := right.(float64); ok {
			switch {
			case leftValue < rightValue:
				return -1
			case leftValue > rightValue:
				return 1
			default:
				return 0
			}
		}
	case string:
		if rightValue, ok := right.(string); ok {
			return strings.Compare(leftValue, rightValue)
		}
	}
	return 0
}

func selectKeys(rows []map[string]any, keys []string) []map[string]any {
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		picked := make(map[string]any, len(keys)+1)
		if id, ok := row["id"]; ok {
			picked["id"] = id
		}
		for _, key := range keys {
			if value, ok := row[key]; ok {
				picked[key] = value
			}
		}
		out[i] = picked
	}
	return out
}

func filterRows(rows []map[string]any, match func(row map[string]any) bool) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if match(row) {
			out = append(out, row)
		}
	}
	return out
}

// Look in the title and the description. If either one contains the search
// words (ignoring uppercase and lowercase), say yes. Otherwise say no.
func matchesSearch(row map[string]any, query string) bool {
	needle := strings.ToLower(query)
	for _, field := range []string{"title", "description"} {
		text, ok := row[field].(string)
		if ok && strings.Contains(strings.ToLower(text), needle) {
			return true
		}
	}
	return false
}

// Walk the key one dot at a time. Open a map, read the part, and use that as
// the next step. If a step is missing or is not a map, say no.
func lookupPath(row map[string]any, path string) (any, bool) {
	var current any = row
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		current, ok = object[part]

		if !ok {
			return nil, false
		}
	}

	return current, true
}

// Find the value at the key path. If it is a string and equals the wanted
// value exactly, say yes. Otherwise say no.
func matchesFilter(row map[string]any, key, value string) bool {
	found, ok := lookupPath(row, key)
	if !ok {
		return false
	}

	text, ok := found.(string)
	if !ok {
		return false
	}

	return text == value
}

func filterByID(rows []map[string]any, key string, id int) []map[string]any {
	return filterRows(rows, func(row map[string]any) bool {
		value, ok := row[key].(float64)
		return ok && int(value) == id
	})
}
