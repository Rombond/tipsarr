// The score shown on posters. Everyone picks a source in their profile; only movies have anything
// besides the TMDB score (read through Radarr), so shows, and movies Radarr has no score for, keep TMDB.
import { api, unwrap, type Schemas } from '$lib/api/client';

export type RatingSource = 'tmdb' | 'imdb' | 'metacritic' | 'rottenTomatoes';
export type Scores = Schemas['MovieScores'];

export const RATING_SOURCES: { value: RatingSource; label: string }[] = [
	{ value: 'tmdb', label: 'TMDB' },
	{ value: 'imdb', label: 'IMDb' },
	{ value: 'metacritic', label: 'Metacritic' },
	{ value: 'rottenTomatoes', label: 'Rotten Tomatoes' },
];

export type Shown = { label: string; tone: string; text: string };

/** The score of one source for display, or null when there is none. */
export function pick(source: RatingSource, scores: Scores | undefined): Shown | null {
	switch (source) {
		case 'imdb':
			return scores?.imdb ? { label: 'IMDb', tone: 'bg-[#f5c518] text-black', text: scores.imdb.value.toFixed(1) } : null;
		case 'metacritic': {
			const v = scores?.metacritic?.value;
			return v ? { label: 'MC', tone: `${v >= 61 ? 'bg-emerald-600' : v >= 40 ? 'bg-amber-500' : 'bg-red-600'} text-white`, text: String(Math.round(v)) } : null;
		}
		case 'rottenTomatoes': {
			const v = scores?.rottenTomatoes?.value;
			return v ? { label: 'RT', tone: `${v >= 60 ? 'bg-red-600' : 'bg-emerald-700'} text-white`, text: `${Math.round(v)}%` } : null;
		}
		default:
			return null;
	}
}

class Ratings {
	/** TMDB id -> scores ({} = Radarr has none) */
	scores = $state<Record<number, Scores>>({});
	private asked = new Set<number>();
	private queue = new Set<number>();
	private timer: ReturnType<typeof setTimeout> | null = null;

	/** Ask for a movie's scores; many calls in a short time become one request. */
	want(id: number) {
		if (this.asked.has(id)) return;
		this.asked.add(id);
		this.queue.add(id);
		this.timer ??= setTimeout(() => this.flush(), 80);
	}

	private async flush() {
		this.timer = null;
		const ids = [...this.queue];
		this.queue.clear();
		for (let i = 0; i < ids.length; i += 40) {
			const chunk = ids.slice(i, i + 40);
			try {
				const res = await unwrap(api.GET('/ratings/movies', { params: { query: { ids: chunk.join(',') } } }));
				for (const id of chunk) {
					const s = res[String(id)];
					if (s) this.scores[id] = s;
					else this.asked.delete(id); // not ready yet: a later look asks again
				}
			} catch {
				for (const id of chunk) this.asked.delete(id);
			}
		}
	}
}

export const ratings = new Ratings();
