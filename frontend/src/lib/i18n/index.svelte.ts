// Tiny typed i18n runtime. Catalogs are plain TypeScript objects: `en` defines the keys, every
// other language must provide exactly the same keys (the compiler enforces it), so a missing
// translation can never ship.
import { en } from './en';
import { fr } from './fr';

export type Key = keyof typeof en;
export type Locale = 'en' | 'fr';
type Catalog = Record<Key, string>;

export const LOCALES: { code: Locale; name: string; tag: string }[] = [
	{ code: 'en', name: 'English', tag: 'en-US' },
	{ code: 'fr', name: 'Français', tag: 'fr-FR' },
];

const catalogs: Record<Locale, Catalog> = { en, fr };
const STORAGE_KEY = 'tipsarr-lang';

function supported(code: string | null | undefined): Locale | null {
	const base = (code ?? '').toLowerCase().slice(0, 2);
	return LOCALES.find((l) => l.code === base)?.code ?? null;
}

class I18n {
	locale = $state<Locale>('en');
	/** the admin's app-wide default (a TMDB tag such as fr-FR), if any */
	private appDefault: Locale | null = null;

	/** BCP 47 tag for Intl and for TMDB (e.g. fr-FR). */
	get tag() {
		return LOCALES.find((l) => l.code === this.locale)!.tag;
	}

	/** Pick the starting language: saved choice, then the app default, then the browser; the profile overrides it after login. */
	init() {
		let saved: string | null = null;
		try {
			saved = localStorage.getItem(STORAGE_KEY);
		} catch {
			/* private mode */
		}
		const browser = typeof navigator !== 'undefined' ? navigator.languages?.map(supported).find(Boolean) : null;
		this.apply(supported(saved) ?? browser ?? 'en');
	}

	/** The app-wide default language: it applies unless the person chose one (login switcher or profile). */
	setAppDefault(tag: string | null | undefined) {
		this.appDefault = supported(tag);
		let saved: string | null = null;
		try {
			saved = localStorage.getItem(STORAGE_KEY);
		} catch {
			/* private mode */
		}
		if (!supported(saved) && this.appDefault) this.apply(this.appDefault);
	}

	/** Use the account's language (profile) once known; empty means "keep the current one". */
	useProfileLanguage(lang: string | null | undefined) {
		const l = supported(lang);
		if (l) this.apply(l);
	}

	/** Forget the explicit choice: follow the app default, or the browser when there is none. */
	useBrowser() {
		try {
			localStorage.removeItem(STORAGE_KEY);
		} catch {
			/* ignore */
		}
		const browser = typeof navigator !== 'undefined' ? navigator.languages?.map(supported).find(Boolean) : null;
		this.apply(this.appDefault ?? browser ?? 'en');
	}

	set(locale: Locale) {
		try {
			localStorage.setItem(STORAGE_KEY, locale);
		} catch {
			/* ignore */
		}
		this.apply(locale);
	}

	private apply(locale: Locale) {
		this.locale = locale;
		if (typeof document !== 'undefined') document.documentElement.lang = locale;
	}
}

export const i18n = new I18n();

/**
 * Translate a key. `{name}` placeholders are replaced from `params`; a message with forms
 * separated by " | " is a plural: the first form is used for "one", the second otherwise
 * (chosen from `params.count` with Intl.PluralRules, so French counts 0 and 1 as singular).
 */
export function t(key: Key, params?: Record<string, string | number>): string {
	let msg: string = catalogs[i18n.locale][key] ?? en[key] ?? key;
	if (msg.includes(' | ')) {
		const forms = msg.split(' | ');
		const count = Number(params?.count ?? 0);
		const form = new Intl.PluralRules(i18n.tag).select(count) === 'one' ? 0 : 1;
		msg = forms[Math.min(form, forms.length - 1)];
	}
	if (params) msg = msg.replace(/\{(\w+)\}/g, (m, k) => (k in params ? String(params[k]) : m));
	return msg;
}

/** True if the key exists (used to fall back to the server's message for unknown error codes). */
export function hasKey(key: string): key is Key {
	return key in en;
}
