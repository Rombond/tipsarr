# Translating Tipsarr

Tipsarr speaks English and French today, and any other language can be added by editing plain JSON files. You do not need to know the code, and a language does not have to be finished to be useful: whatever you leave out is shown in English.

## Where the files are

```
locales/
  languages.json      the list of languages
  en/                 English, the source of all texts
    web.json          the web app
    mobile.json       the iOS and Android apps
    android.json      Android wording that differs from iOS (a few lines)
    ios.json          iOS permission prompts (one line)
    server.json       texts the server sends: push notification titles and background-job summaries
  fr/                 French, same files
```

Each file is a flat list of `"key": "text"` lines. The key never changes, only the text after it.

## Add a language

1. Pick the language code (`es`, `de`, `pt-BR`, ...). Copy the folder `locales/en/` to `locales/<code>/`.
2. Add one line to `locales/languages.json`: the code, the language's name written in that language, and its TMDB tag (the tag decides the language of titles and posters, for example `es-ES`).

   ```json
   { "code": "es", "name": "Español", "tag": "es-ES" }
   ```
3. Translate the texts in your copies (the values after the colon). You may delete the lines you do not want to translate yet.
4. Run `node design/check-locales.mjs`. It lists how much is translated and stops on real mistakes.
5. Open a pull request. A maintainer runs `make generate` and `make ios-strings`, which build the Android, iOS and server files from your JSON; nothing else needs to change.

## Rules

- Keep the placeholders exactly as they are, written `{name}` (for example `"Hello {user}"`). Move them where your language needs them, but do not translate or remove them.
- Plurals use two forms separated by ` | `: `"{count} item | {count} items"`. Keep two forms. The first form is used for one, the second for every other number. (Languages with more plural forms are not supported yet; open an issue.)
- `mobile.json` texts use `%@` or `%1$@` in a few older lines: keep them as they are.
- Do not add keys that English does not have, and do not rename keys. To add a new text, add it to `locales/en/` first.
- Keep texts short: they sit in buttons and small screens.
- Keep the product names as they are: Tipsarr, Jellyfin, Radarr, Sonarr, TMDB, Face ID.
- Save the files as UTF-8 and keep the JSON valid (the checker tells you where it is not).

## Check your work

```
node design/check-locales.mjs
```

prints one line per language with how many texts are translated in each file, then lists mistakes:

- a key English does not have,
- a placeholder that was changed, added or removed,
- a different number of plural forms,
- an empty text (delete the line instead).

Missing translations are only listed, they do not fail the check. The web checks also run in the continuous integration of every pull request.

To see the web app in your language: `cd frontend && npm run dev`, then choose the language on the sign-in page.

## For maintainers

- `make generate` writes `android/app/src/main/res/values-<code>/strings.xml`, `GeneratedLanguages.kt` (the language list of the Android app) and `backend/internal/i18n/locales/<code>.json` (embedded in the server).
- The server sends some texts as a code plus values (for example the summary of a finished sync, `srv.job.library.synced` with the number of movies). The apps and the web app translate the code from the same `server.json`, so translators only edit that file. A new server text is added to `locales/en/server.json` and sent with `i18n.M(code, name, value, ...)` in Go.
- `make ios-strings` copies the string catalogs into the iOS project and adds the language to the project's list of languages.
- The web app reads `locales/<code>/web.json` directly; nothing to generate.
