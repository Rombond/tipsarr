// Package i18n holds the texts the server writes itself: push alert titles and the summaries of background jobs.
// The texts live in locales/<code>/server.json at the repository root (`make generate` copies them to locales/
// here so the server embeds them). The server sends a code plus parameters; the web and mobile apps translate
// the same keys from their own catalogs. The English text is only the fallback for logs and for clients that
// do not know a code.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed locales/*.json
var files embed.FS

// catalogs maps a lower-case language code (fr, pt-br) to code -> text.
var catalogs = load()

func load() map[string]map[string]string {
	out := map[string]map[string]string{}
	entries, err := files.ReadDir("locales")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		raw, err := files.ReadFile("locales/" + e.Name())
		if err != nil {
			panic(err)
		}
		texts := map[string]string{}
		if err := json.Unmarshal(raw, &texts); err != nil {
			panic(fmt.Sprintf("i18n locales/%s: %v", e.Name(), err))
		}
		out[strings.ToLower(strings.TrimSuffix(e.Name(), ".json"))] = texts
	}
	return out
}

// Msg is a message for people: a code that names a text in server.json and the values for its {placeholders}.
type Msg struct {
	Code   string         `json:"code"`
	Params map[string]any `json:"params,omitempty"`
}

// M builds a message from a code and name, value pairs: M("srv.job.library.synced", "movies", 12, "shows", 3).
func M(code string, kv ...any) Msg {
	m := Msg{Code: code}
	for i := 0; i+1 < len(kv); i += 2 {
		if m.Params == nil {
			m.Params = map[string]any{}
		}
		m.Params[fmt.Sprint(kv[i])] = kv[i+1]
	}
	return m
}

func (m Msg) IsZero() bool { return m.Code == "" }

// OrderedArgs are the values in the order the {placeholders} first appear in the English text: the apps
// format the translated text with positional arguments (%1$@, %1$s), numbered in that order.
func (m Msg) OrderedArgs() []string {
	text, ok := catalogs["en"][m.Code]
	if !ok || len(m.Params) == 0 {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		end := strings.IndexByte(text[i:], '}')
		if end < 0 {
			break
		}
		name := text[i+1 : i+end]
		if !seen[name] {
			seen[name] = true
			out = append(out, fmt.Sprint(m.Params[name]))
		}
		i += end
	}
	return out
}

// StringParams are the values as text, the form the API sends them in.
func (m Msg) StringParams() map[string]string {
	if len(m.Params) == 0 {
		return nil
	}
	out := make(map[string]string, len(m.Params))
	for k, v := range m.Params {
		out[k] = fmt.Sprint(v)
	}
	return out
}

// String is the English text (for logs and as the fallback shown by a client that does not know the code).
func (m Msg) String() string {
	if m.IsZero() {
		return ""
	}
	text, _ := T("en", m.Code, m.Params)
	return text
}

// T renders a code in a language (fr, fr-FR, pt-BR ...): the exact tag, then the language, then English.
// It reports false when no language knows the code.
func T(lang, code string, params map[string]any) (string, bool) {
	l := strings.ToLower(strings.ReplaceAll(lang, "_", "-"))
	candidates := []string{l}
	if base, _, ok := strings.Cut(l, "-"); ok {
		candidates = append(candidates, base)
	}
	candidates = append(candidates, "en")
	for _, c := range candidates {
		if text, ok := catalogs[c][code]; ok {
			return fill(text, params), true
		}
	}
	return "", false
}

func fill(text string, params map[string]any) string {
	if len(params) == 0 {
		return text
	}
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			b.WriteByte(text[i])
			continue
		}
		end := strings.IndexByte(text[i:], '}')
		if end < 0 {
			b.WriteString(text[i:])
			break
		}
		name := text[i+1 : i+end]
		if v, ok := params[name]; ok {
			fmt.Fprint(&b, v)
		} else {
			b.WriteString(text[i : i+end+1])
		}
		i += end
	}
	return b.String()
}
