// Reads and checks the translation files in locales/. Shared by build-strings.mjs and check-locales.mjs.
//
// locales/languages.json lists the languages (code, name in its own language, TMDB tag). Each language has a folder
// locales/<code>/ with up to five plain JSON files: web.json (web app), mobile.json (iOS and Android),
// android.json (Android wording that differs from iOS), ios.json (iOS permission prompts) and server.json (push alert titles).
// English is the source: a language may leave keys out (the text falls back to English) but may not add keys
// English does not have, and every translation must keep the same {placeholders} and plural forms.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const root = path.join(path.dirname(fileURLToPath(import.meta.url)), '..');
export const localesDir = path.join(root, 'locales');
export const AREAS = ['web', 'mobile', 'android', 'ios', 'server'];

const readJson = (file, fallback) => {
  if (!fs.existsSync(file)) return fallback;
  try {
    return JSON.parse(fs.readFileSync(file, 'utf8'));
  } catch (e) {
    throw new Error(`${path.relative(root, file)}: not valid JSON (${e.message})`);
  }
};

export function loadLocales() {
  const languages = readJson(path.join(localesDir, 'languages.json'), []);
  if (languages[0]?.code !== 'en') throw new Error('locales/languages.json must list English first');
  const byCode = {};
  for (const l of languages) {
    byCode[l.code] = {};
    for (const area of AREAS) byCode[l.code][area] = readJson(path.join(localesDir, l.code, `${area}.json`), {});
  }
  return { languages, byCode };
}

const vars = (s) => [...new Set([...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]))].sort().join(',');
const forms = (s) => s.split(' | ').length;

/** Problems that break a build (errors) and gaps that do not (missing), per language and area. */
export function validate({ languages, byCode }) {
  const errors = [];
  const coverage = {};
  for (const l of languages) {
    if (!/^[a-z]{2,3}(-[A-Z]{2})?$/.test(l.code)) errors.push(`languages.json: "${l.code}" is not a language code like fr or pt-BR`);
    if (!l.name || !l.tag) errors.push(`languages.json: ${l.code} needs a name and a tag`);
    if (l.code !== 'en' && !fs.existsSync(path.join(localesDir, l.code))) errors.push(`locales/${l.code}/ does not exist`);
  }
  for (const dir of fs.readdirSync(localesDir, { withFileTypes: true })) {
    if (dir.isDirectory() && !languages.some((l) => l.code === dir.name)) errors.push(`locales/${dir.name}/ is not listed in languages.json`);
  }
  for (const l of languages) {
    coverage[l.code] = {};
    for (const area of AREAS) {
      const base = byCode.en[area];
      const cat = byCode[l.code][area];
      coverage[l.code][area] = { have: 0, total: Object.keys(base).length };
      for (const [k, v] of Object.entries(cat)) {
        const where = `locales/${l.code}/${area}.json: ${k}`;
        if (typeof v !== 'string') { errors.push(`${where} is not a string`); continue; }
        if (!(k in base)) { if (l.code !== 'en') errors.push(`${where} does not exist in English (remove it, or add it to locales/en first)`); continue; }
        coverage[l.code][area].have++;
        if (l.code === 'en') continue;
        if (v.trim() === '') errors.push(`${where} is empty (delete the line to fall back to English)`);
        else if (vars(base[k]) !== vars(v)) errors.push(`${where} placeholders differ (English has {${vars(base[k])}}, this has {${vars(v)}})`);
        else if (forms(base[k]) !== forms(v)) errors.push(`${where} has ${forms(v)} plural form(s), English has ${forms(base[k])} (forms are separated by " | ")`);
      }
    }
  }
  return { errors, coverage };
}
