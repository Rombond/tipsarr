import CheckIcon from '@lucide/svelte/icons/check';
import CircleDashedIcon from '@lucide/svelte/icons/circle-dashed';
import ClockIcon from '@lucide/svelte/icons/clock';
import ThumbsUpIcon from '@lucide/svelte/icons/thumbs-up';
import SearchIcon from '@lucide/svelte/icons/search';
import DownloadIcon from '@lucide/svelte/icons/download';
import BanIcon from '@lucide/svelte/icons/ban';
import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';

/** Every state a title can be in, for posters, request rows and detail pages. One colour each. */
export type StatusKey = 'available' | 'partial' | 'requested' | 'approved' | 'searching' | 'downloading' | 'declined' | 'failed';

export const STATUS: Record<StatusKey, { icon: typeof CheckIcon; solid: string; soft: string }> = {
	available: { icon: CheckIcon, solid: 'bg-emerald-500 text-white', soft: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300' },
	partial: { icon: CircleDashedIcon, solid: 'bg-teal-500 text-white', soft: 'bg-teal-500/15 text-teal-700 dark:text-teal-300' },
	requested: { icon: ClockIcon, solid: 'bg-amber-500 text-white', soft: 'bg-amber-500/15 text-amber-700 dark:text-amber-300' },
	approved: { icon: ThumbsUpIcon, solid: 'bg-sky-500 text-white', soft: 'bg-sky-500/15 text-sky-700 dark:text-sky-300' },
	searching: { icon: SearchIcon, solid: 'bg-fuchsia-500 text-white', soft: 'bg-fuchsia-500/15 text-fuchsia-700 dark:text-fuchsia-300' },
	downloading: { icon: DownloadIcon, solid: 'bg-indigo-500 text-white', soft: 'bg-indigo-500/15 text-indigo-700 dark:text-indigo-300' },
	declined: { icon: BanIcon, solid: 'bg-rose-500 text-white', soft: 'bg-rose-500/15 text-rose-700 dark:text-rose-300' },
	failed: { icon: TriangleAlertIcon, solid: 'bg-orange-600 text-white', soft: 'bg-orange-500/15 text-orange-700 dark:text-orange-300' },
};

/** The state a MediaItem shows: availability wins, then the active request. */
export function itemStatus(item: { availability: string; requestStatus?: string | null }, extra?: { tracked?: boolean; hasFile?: boolean }): StatusKey | null {
	if (item.availability === 'available') return 'available';
	if (item.availability === 'partial') return 'partial';
	if (item.requestStatus === 'pending') return 'requested';
	if (item.requestStatus === 'approved') return 'approved';
	if (extra?.tracked) return extra.hasFile ? 'available' : 'approved'; // in Radarr/Sonarr: same as an approved request (or already downloaded)
	return null;
}
