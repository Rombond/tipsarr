// Light / dark / system theme, persisted in localStorage (applied early by app.html).
type Theme = 'light' | 'dark' | 'system';

const KEY = 'tipsarr-theme';

function read(): Theme {
	try {
		const v = localStorage.getItem(KEY);
		if (v === 'light' || v === 'dark' || v === 'system') return v;
	} catch {
		/* private mode */
	}
	return 'system';
}

function apply(t: Theme) {
	const dark = t === 'dark' || (t === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
	document.documentElement.classList.toggle('dark', dark);
}

class ThemeState {
	value = $state<Theme>('system');

	init() {
		this.value = read();
		apply(this.value);
		matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
			if (this.value === 'system') apply('system');
		});
	}

	set(t: Theme) {
		this.value = t;
		try {
			localStorage.setItem(KEY, t);
		} catch {
			/* ignore */
		}
		apply(t);
	}
}

export const theme = new ThemeState();
