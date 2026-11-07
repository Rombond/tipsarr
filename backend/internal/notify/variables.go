package notify

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/i18n"
)

// Variable is one template variable as the editor help shows it.
type Variable struct {
	Path         string `json:"path" doc:"Liquid path, e.g. media.title"`
	Type         string `json:"type" enum:"string,number,boolean" doc:"Value type"`
	Example      string `json:"example" doc:"Value with the sample data (empty when Empty)"`
	Description  string `json:"description" doc:"Localized description"`
	Empty        bool   `json:"empty" doc:"Not filled for this event"`
	PerRecipient bool   `json:"perRecipient" doc:"Only filled with one delivery per recipient"`
}

// Variables flattens template variables. all gives every path (vars of the richest sample event, per
// recipient); vars are the values for the chosen event. Both come from Vars, so the list follows deliveries.
func Variables(all, vars map[string]any, lang string) []Variable {
	got := map[string]any{}
	flatten("", vars, got)
	paths := map[string]any{}
	flatten("", all, paths)
	out := make([]Variable, 0, len(paths))
	for p, ref := range paths {
		v := Variable{Path: p, Type: typeOf(ref), PerRecipient: strings.HasPrefix(p, "notifyUser.")}
		val, ok := got[p]
		if !ok || val == nil || val == "" {
			v.Empty = true
		} else if s, isStr := val.(string); isStr {
			v.Example = s
		} else {
			v.Example = fmt.Sprint(val)
		}
		if t, ok := i18n.T(lang, "webhook.var."+p, nil); ok {
			v.Description = t
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func flatten(prefix string, v any, out map[string]any) {
	m, ok := v.(map[string]any)
	if !ok {
		out[prefix] = v
		return
	}
	for k, x := range m {
		p := k
		if prefix != "" {
			p = prefix + "." + k
		}
		flatten(p, x, out)
	}
}

func typeOf(v any) string {
	switch v.(type) {
	case bool:
		return "boolean"
	case int, int64, float64, json.Number:
		return "number"
	}
	return "string"
}
