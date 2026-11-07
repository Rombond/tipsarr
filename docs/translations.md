# Translations

Tipsarr is available in English and French. The web interface, the server messages and both apps share the same texts:
one folder per language in [`locales/`](../locales), plain JSON files. A missing translation falls back to English.

How to add or fix a language: [CONTRIBUTING-TRANSLATIONS.md](../CONTRIBUTING-TRANSLATIONS.md).

> [!WARNING]
> Plural forms are separated by ` | ` (space, pipe, space): `"{count} issue | {count} issues"`. Any string containing
> ` | ` is split into plural forms, including a literal Liquid example such as `{{ message | json }}`. Put such text
> in a placeholder (`{example}`) filled in by the code, as the webhook hints do, never directly in the translation.

`make locales-check` (or `npm run check:i18n` in `frontend/`) checks placeholders and plural forms; CI runs it.
