// Verifies the translation catalogs beyond what the compiler checks:
// every language has the same keys, the same {placeholders} and the same number of plural forms.
import { readFileSync, readdirSync } from 'node:fs';

const dir = new URL('../src/lib/i18n/', import.meta.url);

function load(file) {
	const map = new Map();
	for (const line of readFileSync(new URL(file, dir), 'utf8').split('\n')) {
		const m = /^\t("(?:[^"\\]|\\.)*"):\s*("(?:[^"\\]|\\.)*"),?$/.exec(line);
		if (m) map.set(JSON.parse(m[1]), JSON.parse(m[2]));
	}
	return map;
}

const vars = (s) => [...new Set([...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]))].sort().join(',');
const forms = (s) => s.split(' | ').length;

const base = load('en.ts');
let problems = 0;
const fail = (msg) => {
	console.error('i18n: ' + msg);
	problems++;
};

for (const file of readdirSync(dir)) {
	if (!/^[a-z]{2}\.ts$/.test(file) || file === 'en.ts') continue;
	const cat = load(file);
	for (const [k, en] of base) {
		const tr = cat.get(k);
		if (tr === undefined) {
			fail(`${file}: missing key ${k}`);
			continue;
		}
		if (vars(en) !== vars(tr)) fail(`${file}: ${k} placeholders differ (en {${vars(en)}} vs {${vars(tr)}})`);
		if (forms(en) !== forms(tr)) fail(`${file}: ${k} has a different number of plural forms`);
		if (tr.trim() === '') fail(`${file}: ${k} is empty`);
	}
	for (const k of cat.keys()) if (!base.has(k)) fail(`${file}: extra key ${k}`);
	console.log(`${file}: ${cat.size} keys checked against ${base.size}`);
}

if (problems) process.exit(1);
console.log('i18n catalogs are consistent');
