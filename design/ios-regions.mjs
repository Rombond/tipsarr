// Keeps the iOS project's list of supported languages in step with locales/languages.json.
// Run by `make ios-strings`. Xcode only shows a language in the app if the project lists it.
import fs from 'node:fs';
import path from 'node:path';
import { loadLocales, root } from './locales.mjs';

const file = path.join(root, 'ios', 'Tipsarr', 'Tipsarr.xcodeproj', 'project.xcproj');
const { languages } = loadLocales();
const wanted = ['Base', ...languages.map((l) => l.code).filter((c) => c !== 'en')];
const src = fs.readFileSync(file, 'utf8');
const re = /("supported": \[\n)((?:\s+"[^"]+",\n)+)(\s+\],)/;
const m = re.exec(src);
if (!m) { console.error('ios-regions: cannot find the "supported" language list in project.xcproj'); process.exit(1); }
const list = wanted.map((c) => `      "${c}",\n`).join('');
if (m[2] === list) { console.log('ios-regions: project already lists ' + wanted.join(', ')); process.exit(0); }
fs.writeFileSync(file, src.replace(re, `$1${list}$3`));
console.log('ios-regions: project now lists ' + wanted.join(', '));
