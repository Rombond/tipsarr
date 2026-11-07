package main

import (
	"html/template"
	"net/http"
	"strings"
)

var pageTmpl = template.Must(template.New("p").Parse(`<!doctype html>
<html lang="{{.Lang}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="refresh" content="60"><title>Tipsarr demo</title>
<style>
:root{--bg:#f7f7f8;--fg:#18181b;--muted:#71717a;--card:#fff;--accent:#6d28d9}
@media (prefers-color-scheme:dark){:root{--bg:#09090b;--fg:#fafafa;--muted:#a1a1aa;--card:#18181b;--accent:#a78bfa}}
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:var(--bg);color:var(--fg);font-family:system-ui,sans-serif;padding:0 16px}
main{max-width:28rem;background:var(--card);border-radius:12px;padding:2rem;box-shadow:0 1px 3px rgba(0,0,0,.1)}
h1{font-size:1.25rem;margin:0 0 .75rem}p{color:var(--muted);line-height:1.5}a{color:var(--accent)}
</style></head><body><main><h1>{{.Title}}</h1><p>{{.Text}}</p><p><a href="/">{{.Retry}}</a></p></main></body></html>`))

type pageText struct{ Lang, Title, Text, Retry string }

var pages = map[string]map[string]pageText{
	"full": {
		"en": {"en", "The demo is full", "Every sandbox is in use right now. Each one lasts an hour at most, so please try again in a few minutes.", "Try again"},
		"fr": {"fr", "La démo est pleine", "Tous les bacs à sable sont occupés pour le moment. Chacun dure une heure au plus : réessaie dans quelques minutes.", "Réessayer"},
	},
	"ip": {
		"en": {"en", "Too many demos", "You have opened many demo sessions in a short time. Please wait a little before starting a new one.", "Try again"},
		"fr": {"fr", "Trop de démos", "Tu as ouvert beaucoup de sessions de démo en peu de temps. Attends un peu avant d’en lancer une nouvelle.", "Réessayer"},
	},
}

// lang picks French when the browser prefers it, English otherwise.
func lang(r *http.Request) string {
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		switch {
		case strings.HasPrefix(tag, "fr"):
			return "fr"
		case strings.HasPrefix(tag, "en"):
			return "en"
		}
	}
	return "en"
}

func page(w http.ResponseWriter, r *http.Request, status int, kind string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", "300")
	w.WriteHeader(status)
	_ = pageTmpl.Execute(w, pages[kind][lang(r)])
}
