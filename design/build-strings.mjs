// Builds the mobile string catalogs from the web catalogs + design/strings/mobile.*.json.
// Run: node design/build-strings.mjs   Outputs design/generated/strings.json and Localizable.xcstrings
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const i18n = path.join(here, '..', 'frontend', 'src', 'lib', 'i18n');
const out = path.join(here, 'generated');
fs.mkdirSync(out, { recursive: true });

// web catalogs are `export const xx = { "key": "value", ... }` literals
const readCatalog = (file, name) => {
  const src = fs.readFileSync(path.join(i18n, file), 'utf8');
  const m = src.match(new RegExp(`export const ${name}[^=]*=\\s*(\\{[\\s\\S]*\\})\\s*(as const)?;?\\s*$`));
  if (!m) throw new Error(`cannot read ${file}`);
  return new Function(`return (${m[1]})`)();
};
const en = readCatalog('en.ts', 'en');
const fr = readCatalog('fr.ts', 'fr');

// namespaces that only exist on the admin web UI are left out of the apps
const webOnly = new Set(['settings', 'webhooks', 'services', 'setup', 'job', 'box', 'boxrow', 'checklist', 'form', 'errpage', 'brand', 'modal', 'carousel']);
const keep = (k) => !webOnly.has(k.split('.')[0]);
const mobile = { en: JSON.parse(fs.readFileSync(path.join(here, 'strings', 'mobile.en.json'), 'utf8')), fr: JSON.parse(fs.readFileSync(path.join(here, 'strings', 'mobile.fr.json'), 'utf8')) };

const pick = (cat) => Object.fromEntries(Object.entries(cat).filter(([k]) => keep(k)));
const strings = { en: { ...pick(en), ...mobile.en }, fr: { ...pick(fr), ...mobile.fr } };

const missing = [];
for (const k of Object.keys(strings.en)) if (!(k in strings.fr)) missing.push(`fr is missing ${k}`);
for (const k of Object.keys(strings.fr)) if (!(k in strings.en)) missing.push(`en is missing ${k}`);
const ph = (s) => (s.match(/\{(\w+)\}/g) || []).map((x) => x.slice(1, -1)).sort().join();
for (const k of Object.keys(strings.en)) if (strings.fr[k] !== undefined && ph(strings.en[k]) !== ph(strings.fr[k])) missing.push(`placeholders differ for ${k}`);
if (missing.length) { console.error(missing.join('\n')); process.exit(1); }

fs.writeFileSync(path.join(out, 'strings.json'), JSON.stringify(strings, null, 1) + '\n');

// Xcode String Catalog: {name} placeholders become positional %n$@ in order of appearance (the order is kept in the comment)
// A literal % next to placeholders is written %% (a lone % at the end of a format is dropped).
const toPos = (s, order) => { const names = []; const text = (/\{\w+\}/.test(s) ? s.replace(/%/g, '%%') : s).replace(/\{(\w+)\}/g, (_, nm) => { names.push(nm); return `%${(order || names).indexOf(nm) + 1}$@`; }); return { text: names.length === 1 ? text.replace('%1$@', '%@') : text, names }; };
const catalog = { sourceLanguage: 'en', version: '1.0', strings: {} };
for (const k of Object.keys(strings.en).sort()) {
  const e = toPos(strings.en[k]); const f = toPos(strings.fr[k], e.names);
  catalog.strings[k] = { ...(e.names.length ? { comment: `placeholders: ${e.names.join(', ')}` } : {}), localizations: { en: { stringUnit: { state: 'translated', value: e.text } }, fr: { stringUnit: { state: 'translated', value: f.text } } } };
}
fs.writeFileSync(path.join(out, 'Localizable.xcstrings'), JSON.stringify(catalog, null, 2) + '\n');
console.log(`wrote ${Object.keys(strings.en).length} strings (web ${Object.keys(pick(en)).length}, mobile ${Object.keys(mobile.en).length})`);
