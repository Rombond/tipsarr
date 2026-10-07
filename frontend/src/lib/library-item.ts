import type { MediaItem } from '$lib/api/client';

/** A title of the Library or Stats pages as the card used everywhere else expects it. Everything listed is in Jellyfin. */
export function asMediaItem(it: { type: 'movie' | 'tv'; tmdbId: number; title: string; year?: number; rating?: number }): MediaItem {
	return {
		type: it.type,
		tmdbId: it.tmdbId,
		title: it.title,
		releaseDate: it.year ? String(it.year) : undefined,
		voteAverage: it.rating ?? 0,
		availability: 'available',
	} as MediaItem;
}
