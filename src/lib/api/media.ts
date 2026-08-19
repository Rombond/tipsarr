// Normalizes items from boxarr/suggestarr/seerr into one shape the unified
// carousel/modal components can render without knowing the source.

import { posterUrl } from './seerr';

export type MediaSource = 'boxarr' | 'suggestarr' | 'seerr';

export type MediaItem = {
	key: string;
	source: MediaSource;
	mediaType: 'movie' | 'tv';
	tmdbId?: number | null;
	title: string;
	overview?: string;
	posterUrl?: string | null;
	rating?: number;
	releaseDate?: string;
	availabilityStatus?: number | null;
	raw: unknown;
};

export function fromBoxarrMovie(m: any): MediaItem {
	const tmdbId = m.tmdb_id || m.tmdbId || null;
	return {
		key: `boxarr:${m.radarr_id || m.id || m.title}`,
		source: 'boxarr',
		mediaType: 'movie',
		tmdbId,
		title: m.title,
		overview: m.overview,
		posterUrl: null, // resolved by MoviePoster (radarr/tmdb lookup) in the card itself
		releaseDate: m.release_date,
		availabilityStatus: null,
		raw: m,
	};
}

export function fromSuggestion(s: any): MediaItem {
	return {
		key: `suggestarr:${s.id}`,
		source: 'suggestarr',
		mediaType: s.media_type === 'tv' ? 'tv' : 'movie',
		tmdbId: s.tmdb_id ? Number(s.tmdb_id) : null,
		title: s.title,
		overview: s.overview,
		posterUrl: s.poster_path || null,
		rating: s.rating,
		releaseDate: s.release_date,
		availabilityStatus: null,
		raw: s,
	};
}

export function fromSeerrResult(r: any): MediaItem {
	return {
		key: `seerr:${r.mediaType}:${r.id}`,
		source: 'seerr',
		mediaType: r.mediaType === 'tv' ? 'tv' : 'movie',
		tmdbId: r.id,
		title: r.title || r.name,
		overview: r.overview,
		posterUrl: posterUrl(r.posterPath),
		rating: r.voteAverage,
		releaseDate: r.releaseDate || r.firstAirDate,
		availabilityStatus: r.mediaInfo?.status ?? null,
		raw: r,
	};
}
