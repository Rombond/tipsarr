// Builds every platform's string files from the translation files in locales/ (see design/locales.mjs for the layout).
// Run: node design/build-strings.mjs   (make generate does it)
// Outputs:
//   design/generated/strings.json, Localizable.xcstrings, InfoPlist.xcstrings    iOS (copied into the app by `make ios-strings`)
//   design/generated/android/values[-xx]/strings.xml, GeneratedLanguages.kt       Android (copied by `make generate`)
//   backend/internal/push/locales/<code>.json                                     push alert titles (embedded in the server)
// A language may leave keys out: Android, iOS, the web app and the server then show the English text.
import fs from 'node:fs';
import path from 'node:path';
import { loadLocales, validate, root } from './locales.mjs';

const here = path.dirname(new URL(import.meta.url).pathname);
const out = path.join(here, 'generated');
fs.mkdirSync(out, { recursive: true });

const loc = loadLocales();
const { errors, coverage } = validate(loc);
if (errors.length) { console.error(errors.join('\n')); process.exit(1); }
const codes = loc.languages.map((l) => l.code);

// namespaces that only exist on the admin web UI are left out of the apps
const webOnly = new Set(['settings', 'webhooks', 'services', 'setup', 'job', 'box', 'boxrow', 'checklist', 'form', 'errpage', 'brand', 'modal', 'carousel']);
const keep = (k) => !webOnly.has(k.split('.')[0]);
const pick = (cat) => Object.fromEntries(Object.entries(cat).filter(([k]) => keep(k)));

// the catalog the apps share: web strings that make sense in an app, then the mobile ones (which win)
const strings = Object.fromEntries(codes.map((c) => [c, { ...pick(loc.byCode[c].web), ...loc.byCode[c].mobile }]));
fs.writeFileSync(path.join(out, 'strings.json'), JSON.stringify(strings, null, 1) + '\n');

// Xcode String Catalog: {name} placeholders become positional %n$@ in order of appearance (the order is kept in the comment)
// A literal % next to placeholders is written %% (a lone % at the end of a format is dropped).
const toPos = (s, order) => { const names = []; const text = (/\{\w+\}/.test(s) ? s.replace(/%/g, '%%') : s).replace(/\{(\w+)\}/g, (_, nm) => { names.push(nm); return `%${(order || names).indexOf(nm) + 1}$@`; }); return { text: names.length === 1 ? text.replace('%1$@', '%@') : text, names }; };
const catalog = { sourceLanguage: 'en', version: '1.0', strings: {} };
for (const k of Object.keys(strings.en).sort()) {
  const e = toPos(strings.en[k]);
  const localizations = {};
  for (const c of codes) {
    if (strings[c][k] === undefined) continue;
    localizations[c] = { stringUnit: { state: 'translated', value: c === 'en' ? e.text : toPos(strings[c][k], e.names).text } };
  }
  catalog.strings[k] = { ...(e.names.length ? { comment: `placeholders: ${e.names.join(', ')}` } : {}), localizations };
}
fs.writeFileSync(path.join(out, 'Localizable.xcstrings'), JSON.stringify(catalog, null, 2) + '\n');
console.log(`wrote ${Object.keys(strings.en).length} strings for ${codes.join(', ')}`);

// iOS Info.plist texts (permission prompts) live in locales/<code>/ios.json and become InfoPlist.xcstrings
const infoKeys = Object.keys(loc.byCode.en.ios);
const info = { sourceLanguage: 'en', version: '1.0', strings: {} };
for (const k of infoKeys) {
  info.strings[k] = { localizations: Object.fromEntries(codes.filter((c) => loc.byCode[c].ios[k] !== undefined).map((c) => [c, { stringUnit: { state: 'translated', value: loc.byCode[c].ios[k] } }])) };
}
fs.writeFileSync(path.join(out, 'InfoPlist.xcstrings'), JSON.stringify(info, null, 2) + '\n');

// Android: values/strings.xml (English) and values-<code>/strings.xml per language.
// Key "req.stage.downloading_pct" becomes the resource name req_stage_downloading_pct; {name} placeholders become %n$s.
const xmlName = (k) => k.replace(/[^A-Za-z0-9]+/g, '_').replace(/^(\d)/, 'k$1').toLowerCase();
const androidStrings = Object.fromEntries(codes.map((c) => [c, { ...strings[c], ...loc.byCode[c].android }]));
const seen = new Map();
for (const k of Object.keys(androidStrings.en)) {
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
const androidXml = (code) => {
  const rows = Object.keys(androidStrings.en).sort().filter((k) => androidStrings[code][k] !== undefined).map((k) => {
    const e = xmlValue(androidStrings.en[k]);
    const v = code === 'en' ? e : xmlValue(androidStrings[code][k], e.names);
    return `    <string name="${xmlName(k)}"${v.plain ? ' formatted="false"' : ''}>${v.t}</string>`;
  });
  return `<?xml version="1.0" encoding="utf-8"?>\n<!-- Generated by design/build-strings.mjs from locales/${code}/. Do not edit by hand. -->\n<resources>\n${rows.join('\n')}\n</resources>\n`;
};
// pt-BR -> values-pt-rBR
const androidDir = (code) => (code === 'en' ? 'values' : `values-${code.replace(/-([A-Z]{2})$/, '-r$1')}`);
const androidOut = path.join(out, 'android');
fs.rmSync(androidOut, { recursive: true, force: true });
for (const c of codes) {
  fs.mkdirSync(path.join(androidOut, androidDir(c)), { recursive: true });
  fs.writeFileSync(path.join(androidOut, androidDir(c), 'strings.xml'), androidXml(c));
}
// the in-app language picker lists the languages of locales/languages.json
const kt = loc.languages.map((l) => `        "${l.code}" to "${l.name}"`).join(',\n');
fs.writeFileSync(path.join(androidOut, 'GeneratedLanguages.kt'), `package com.brebond.tipsarr.core.support

// Generated by design/build-strings.mjs from locales/languages.json. Do not edit by hand.
object GeneratedLanguages {
    /** Language code and the language's name in itself. */
    val all: List<Pair<String, String>> = listOf(
${kt},
    )
}
`);
console.log(`wrote Android strings for ${codes.length} language(s)`);

// Server: push alert titles, embedded by backend/internal/push
const pushDir = path.join(root, 'backend', 'internal', 'push', 'locales');
fs.rmSync(pushDir, { recursive: true, force: true });
fs.mkdirSync(pushDir, { recursive: true });
for (const c of codes) fs.writeFileSync(path.join(pushDir, `${c}.json`), JSON.stringify(loc.byCode[c].server, null, 2) + '\n');

// A short coverage line per language, so gaps are visible when generating
for (const c of codes.filter((c) => c !== 'en')) {
  const missing = Object.values(coverage[c]).reduce((n, a) => n + (a.total - a.have), 0);
  if (missing) console.log(`note: ${c} misses ${missing} string(s), English is shown for them (node design/check-locales.mjs lists them)`);
}
