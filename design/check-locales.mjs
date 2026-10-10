// Checks every language in locales/: node design/check-locales.mjs
// Exits with an error on a real problem (unknown key, changed placeholder, wrong plural forms, bad JSON).
// Missing translations are only reported: the text falls back to English.
import { loadLocales, validate, AREAS } from './locales.mjs';

let loc;
try {
  loc = loadLocales();
} catch (e) {
  console.error(e.message);
  process.exit(1);
}
const { errors, coverage } = validate(loc);
for (const l of loc.languages) {
  const parts = AREAS.map((a) => {
    const c = coverage[l.code][a];
    return c.total ? `${a} ${c.have}/${c.total}` : null;
  }).filter(Boolean);
  console.log(`${l.code.padEnd(6)} ${l.name.padEnd(14)} ${parts.join('  ')}`);
}
if (errors.length) {
  console.error('\n' + errors.join('\n'));
  console.error(`\n${errors.length} problem(s)`);
  process.exit(1);
}
console.log('\nlocales are consistent');
