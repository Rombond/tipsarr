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

// Android: strings-en.xml and strings-fr.xml (copied to res/values and res/values-fr by `make generate`).
// Key "req.stage.downloading_pct" becomes the resource name req_stage_downloading_pct; {name} placeholders become %n$s.
const xmlName = (k) => k.replace(/[^A-Za-z0-9]+/g, '_').replace(/^(\d)/, 'k$1').toLowerCase();
const seen = new Map();
for (const k of Object.keys(strings.en)) {
  const n = xmlName(k);
  if (seen.has(n)) { console.error(`android resource name clash: ${k} and ${seen.get(n)}`); process.exit(1); }
  seen.set(n, k);
}
const xmlValue = (s, order) => {
  const names = [];
  // The mobile strings use the iOS placeholders %@ and %1$@; they become %s and %1$s.
  const ios = /%(\d+\$)?@/g;
  const hasPh = /\{\w+\}/.test(s) || ios.test(s);
  let t = s.replace(/%(\d+\$)?@/g, (_, n) => `\u0001${n || ''}s\u0001`);
  if (hasPh) t = t.replace(/%/g, '%%');
  t = t.replace(/\u0001([^\u0001]*)\u0001/g, (_, rest) => `%${rest}`);
  t = t.replace(/\{(\w+)\}/g, (_, nm) => { if (!names.includes(nm)) names.push(nm); return `%${(order || names).indexOf(nm) + 1}$s`; });
  t = t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/'/g, "\\'").replace(/"/g, '\\"').replace(/^([@?])/, '\\$1').replace(/\n/g, '\\n');
  return { t, names, plain: !hasPh && s.includes('%') };
};
// design/strings/android.{en,fr}.json override the shared wording where the iOS text does not fit (App Store, Face ID).
const readOverride = (lang) => { const f = path.join(here, 'strings', `android.${lang}.json`); return fs.existsSync(f) ? JSON.parse(fs.readFileSync(f, 'utf8')) : {}; };
const androidStrings = { en: { ...strings.en, ...readOverride('en') }, fr: { ...strings.fr, ...readOverride('fr') } };
const androidXml = (lang) => {
  const rows = Object.keys(androidStrings.en).sort().map((k) => {
    const e = xmlValue(androidStrings.en[k]);
    const v = lang === 'en' ? e : xmlValue(androidStrings.fr[k], e.names);
    return `    <string name="${xmlName(k)}"${v.plain ? ' formatted="false"' : ''}>${v.t}</string>`;
  });
  return `<?xml version="1.0" encoding="utf-8"?>\n<!-- Generated by design/build-strings.mjs. Do not edit by hand. -->\n<resources>\n${rows.join('\n')}\n</resources>\n`;
};
fs.writeFileSync(path.join(out, 'strings-en.xml'), androidXml('en'));
fs.writeFileSync(path.join(out, 'strings-fr.xml'), androidXml('fr'));
console.log('wrote strings-en.xml and strings-fr.xml');
