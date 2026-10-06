// Locale-aware formatting. Every function reads the current language, so call them from
// templates or $derived values and they re-render when the language changes.
import { hasKey, i18n, t } from './index.svelte';

export function fmtDate(value: string | number | Date | null | undefined, opts: Intl.DateTimeFormatOptions = { dateStyle: 'medium' }): string {
	if (!value) return '';
	const d = typeof value === 'number' ? new Date(value * 1000) : new Date(value);
	return Number.isNaN(d.getTime()) ? '' : new Intl.DateTimeFormat(i18n.tag, opts).format(d);
}

export const fmtDateTime = (value: string | number | Date | null | undefined) => fmtDate(value, { dateStyle: 'medium', timeStyle: 'short' });
export const fmtLongDate = (value: string | null | undefined) => fmtDate(value, { dateStyle: 'long' });

export function fmtNumber(n: number): string {
	return new Intl.NumberFormat(i18n.tag).format(n);
}

/** $32.0M style amounts. The chart is in US dollars. */
export function fmtMoneyCompact(n: number): string {
	if (n <= 0) return '-';
	return new Intl.NumberFormat(i18n.tag, { style: 'currency', currency: 'USD', notation: 'compact', maximumFractionDigits: 1 }).format(n);
}

export function fmtMoney(n: number): string {
	return new Intl.NumberFormat(i18n.tag, { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(n);
}

/** Language name in the UI language, e.g. "ja" -> "Japanese" / "japonais". */
export function languageName(code: string): string {
	try {
		return new Intl.DisplayNames([i18n.tag], { type: 'language' }).of(code) ?? code;
	} catch {
		return code;
	}
}

/** "2026W40" -> the Friday-Sunday range of that ISO week, e.g. "2–4 octobre 2026". */
export function weekendRange(weekKey: string): string {
	const m = /^(\d{4})W(\d{2})$/.exec(weekKey);
	if (!m) return weekKey;
	const year = Number(m[1]);
	const week = Number(m[2]);
	// ISO week 1 contains Jan 4th; Monday of week N, then Friday..Sunday
	const jan4 = new Date(Date.UTC(year, 0, 4));
	const monday = new Date(jan4);
	monday.setUTCDate(jan4.getUTCDate() - ((jan4.getUTCDay() + 6) % 7) + (week - 1) * 7);
	const fri = new Date(monday);
	fri.setUTCDate(monday.getUTCDate() + 4);
	const sun = new Date(monday);
	sun.setUTCDate(monday.getUTCDate() + 6);
	return new Intl.DateTimeFormat(i18n.tag, { dateStyle: 'long', timeZone: 'UTC' }).formatRange(fri, sun);
}

export function fmtDuration(seconds: number, t: (k: 'time.min' | 'time.hours', p: Record<string, number>) => string): string {
	if (seconds >= 3600) return t('time.hours', { h: Math.floor(seconds / 3600), m: Math.round((seconds % 3600) / 60) });
	return t('time.min', { count: Math.max(1, Math.round(seconds / 60)) });
}

/** TMDB department names ("Acting") are English whatever the request language; translate the ones we know. */
export function deptLabel(dept?: string | null): string {
	if (!dept) return '';
	return hasKey(`dept.${dept}`) ? t(`dept.${dept}` as 'dept.Acting') : dept;
}
