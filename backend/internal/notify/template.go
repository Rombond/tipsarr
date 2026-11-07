package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/osteele/liquid"
)

// Templates are Liquid (https://shopify.github.io/liquid/). The variables are listed by Variables (editor help)
// and in docs/notifications.md.
// Values are printed as they are: wrap free text in the json filter ({{ message | json }}) to get a safe,
// quoted JSON string.

var (
	engineOnce sync.Once
	engine     *liquid.Engine
)

func liquidEngine() *liquid.Engine {
	engineOnce.Do(func() {
		engine = liquid.NewEngine()
		engine.RegisterFilter("json", func(v any) string {
			var b bytes.Buffer
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false) // <@id> mentions and & in URLs stay readable
			if err := enc.Encode(v); err != nil {
				return "null"
			}
			return strings.TrimSuffix(b.String(), "\n")
		})
	})
	return engine
}

// TemplateError is a template that does not parse or render. Its text is meant for the admin.
type TemplateError struct {
	Part string // body or url
	Err  error
}

func (e *TemplateError) Error() string { return fmt.Sprintf("%s template: %v", e.Part, e.Err) }
func (e *TemplateError) Unwrap() error { return e.Err }

// CheckTemplate reports a syntax error in a template (rendering errors only show with real values).
func CheckTemplate(part, tpl string) error {
	if !isTemplate(tpl) {
		return nil
	}
	if _, err := liquidEngine().ParseString(tpl); err != nil {
		return &TemplateError{Part: part, Err: err}
	}
	return nil
}

// Render renders a Liquid template with the variables.
func Render(tpl string, vars map[string]any) (string, error) {
	out, err := liquidEngine().ParseAndRenderString(tpl, vars)
	if err != nil {
		return "", err
	}
	return out, nil
}

func isTemplate(s string) bool { return strings.Contains(s, "{{") || strings.Contains(s, "{%") }

// toMap turns a value into plain maps and slices (by its JSON form), so templates see the JSON field names.
// Numbers stay json.Number, which prints without a decimal point and marshals as a number.
func toMap(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var out any
	if d.Decode(&out) != nil {
		return nil
	}
	return out
}
