<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, unwrap, imageUrl, type MediaItem } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import InfoIcon from '@lucide/svelte/icons/info';
	import XIcon from '@lucide/svelte/icons/x';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import CheckIcon from '@lucide/svelte/icons/check';
	import StarIcon from '@lucide/svelte/icons/star';

	let {
		item,
		onSelect,
		rank,
		note,
		inRadarr = false,
		onDismiss,
	}: {
		item: MediaItem;
		onSelect?: (item: MediaItem) => void;
		/** Chart position, shown over the poster. */
		rank?: number;
		/** Small line under the title, e.g. a box-office gross. */
		note?: string;
		/** Radarr already tracks this movie. */
		inRadarr?: boolean;
		/** Shows a "not interested" button on hover (used by suggestion rows). */
		onDismiss?: (item: MediaItem) => void;
	} = $props();

	let requested = $state<string | null>(null);
	let requesting = $state(false);

	const poster = $derived(imageUrl(item.posterPath, 'w342'));
	const year = $derived(item.releaseDate?.slice(0, 4));
	const status = $derived(requested ?? item.requestStatus ?? null);
	const canQuickRequest = $derived(item.type === 'movie' && item.availability !== 'available' && status === null);

	function openDetails() {
		goto(`/media/${item.type}/${item.tmdbId}`);
	}

	async function quickRequest(e: MouseEvent) {
		e.stopPropagation();
		if (requesting) return;
		requesting = true;
		try {
			const r = await unwrap(api.POST('/requests', { body: { type: item.type, tmdbId: item.tmdbId } }));
			requested = r.status;
			toast.success(r.status === 'approved' ? `"${item.title}" approved` : `Requested "${item.title}"`);
		} catch (err) {
			toast.error((err as Error).message);
		} finally {
			requesting = false;
		}
	}
</script>

<div class="w-32 shrink-0 snap-start sm:w-40">
	<div
		class="group relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-muted shadow-sm ring-1 ring-border/50 transition hover:shadow-lg hover:ring-border focus-within:ring-2 focus-within:ring-ring"
	>
		<button
			type="button"
			class="block h-full w-full cursor-pointer text-left"
			aria-label={item.title}
			onclick={() => (onSelect ? onSelect(item) : openDetails())}
		>
			{#if poster}
				<img
					src={poster}
					alt=""
					class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
					loading="lazy"
				/>
			{:else}
				<div class="flex h-full w-full items-center justify-center p-2 text-center text-muted-foreground text-xs">{item.title}</div>
			{/if}
		</button>

		{#if item.availability !== 'none'}
			<Badge class="absolute top-1.5 left-1.5 gap-1 text-[10px]">
				<CheckIcon class="size-3" />
				{item.availability === 'available' ? 'Available' : 'Partial'}
			</Badge>
		{:else if status}
			<Badge variant="secondary" class="absolute top-1.5 left-1.5 text-[10px]">{status === 'pending' ? 'Requested' : 'Approved'}</Badge>
		{:else if inRadarr}
			<Badge variant="secondary" class="absolute top-1.5 left-1.5 text-[10px]">In Radarr</Badge>
		{/if}

		{#if rank}
			<span class="pointer-events-none absolute bottom-1 left-2 font-black text-4xl text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">{rank}</span>
		{/if}

		<!-- hover / focus actions (always reachable by keyboard; on touch the card opens the details) -->
		<div class="absolute inset-x-0 bottom-0 flex items-end justify-between gap-1 bg-gradient-to-t from-black/80 via-black/40 to-transparent p-1.5 pt-8 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:none)]:hidden">
			{#if canQuickRequest}
				<button
					type="button"
					class="flex cursor-pointer items-center gap-1 rounded-full bg-white/90 px-2.5 py-1 text-xs font-medium text-black shadow hover:bg-white disabled:opacity-60"
					disabled={requesting}
					onclick={quickRequest}
				>
					<PlusIcon class="size-3.5" />
					{requesting ? '…' : 'Request'}
				</button>
			{:else}
				<span></span>
			{/if}
			<span class="flex gap-1">
				{#if onDismiss}
					<button
						type="button"
						aria-label="Not interested"
						title="Not interested: hide from my suggestions"
						class="flex size-7 cursor-pointer items-center justify-center rounded-full bg-white/90 text-black shadow hover:bg-white"
						onclick={(e) => {
							e.stopPropagation();
							onDismiss(item);
						}}
					>
						<XIcon class="size-4" />
					</button>
				{/if}
				<button
					type="button"
					aria-label="View details"
					class="flex size-7 cursor-pointer items-center justify-center rounded-full bg-white/90 text-black shadow hover:bg-white"
					onclick={(e) => {
						e.stopPropagation();
						openDetails();
					}}
				>
					<InfoIcon class="size-4" />
				</button>
			</span>
		</div>
	</div>

	<p class="mt-1.5 truncate text-sm font-medium" title={item.title}>{item.title}</p>
	<p class="flex items-center gap-1.5 truncate text-xs text-muted-foreground">
		{#if item.type === 'tv'}<span class="rounded bg-muted px-1 py-px text-[10px] font-medium text-foreground/80">TV</span>{/if}
		{#if year}<span>{year}</span>{/if}
		{#if item.voteAverage}<span class="inline-flex items-center gap-0.5"><StarIcon class="size-3 fill-amber-400 text-amber-400" />{item.voteAverage.toFixed(1)}</span>{/if}
	</p>
	{#if note}<p class="truncate text-xs text-muted-foreground">{note}</p>{/if}
</div>
