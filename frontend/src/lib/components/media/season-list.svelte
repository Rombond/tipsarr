<script lang="ts">
	import { api, unwrap, imageUrl, type Schemas } from '$lib/api/client';
	import ChevronIcon from '@lucide/svelte/icons/chevron-down';

	let { tmdbId, seasons }: { tmdbId: number; seasons: Schemas['Season'][] } = $props();

	let openSeason = $state<number | null>(null);
	let episodes = $state<Record<number, Schemas['Episode'][]>>({});
	let loading = $state<number | null>(null);
	let error = $state<string | null>(null);

	async function toggle(n: number) {
		if (openSeason === n) {
			openSeason = null;
			return;
		}
		openSeason = n;
		if (episodes[n]) return;
		loading = n;
		error = null;
		try {
			const s = await unwrap(api.GET('/media/tv/{id}/seasons/{season}', { params: { path: { id: tmdbId, season: n } } }));
			episodes = { ...episodes, [n]: s.episodes };
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = null;
		}
	}

	const date = (d?: string) => (d ? new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' }) : '');
</script>

<ul class="grid gap-1.5 text-sm">
	{#each seasons as s (s.number)}
		<li class="overflow-hidden rounded-lg border border-border">
			<button
				type="button"
				class="flex w-full cursor-pointer items-center justify-between gap-3 px-3 py-2.5 text-left hover:bg-accent"
				aria-expanded={openSeason === s.number}
				onclick={() => toggle(s.number)}
			>
				<span class="font-medium">{s.name || `Season ${s.number}`}</span>
				<span class="flex items-center gap-2 text-muted-foreground">
					{s.episodeCount} episodes{#if s.airDate} · {s.airDate.slice(0, 4)}{/if}
					<ChevronIcon class="size-4 transition-transform {openSeason === s.number ? 'rotate-180' : ''}" />
				</span>
			</button>
			{#if openSeason === s.number}
				<div class="border-t border-border bg-muted/30 px-3 py-2">
					{#if loading === s.number}
						<p class="py-2 text-muted-foreground">Loading episodes…</p>
					{:else if error}
						<p class="py-2 text-destructive">{error}</p>
					{:else}
						<ol class="grid gap-2">
							{#each episodes[s.number] ?? [] as e (e.number)}
								<li class="flex gap-3">
									<span class="w-6 shrink-0 text-right text-muted-foreground">{e.number}</span>
									<span class="min-w-0">
										<span class="block font-medium">{e.name}</span>
										<span class="block text-xs text-muted-foreground">
											{date(e.airDate)}{#if e.runtimeMinutes} · {e.runtimeMinutes} min{/if}
										</span>
										{#if e.overview}<span class="mt-0.5 line-clamp-2 block text-xs text-muted-foreground">{e.overview}</span>{/if}
									</span>
								</li>
							{/each}
						</ol>
					{/if}
				</div>
			{/if}
		</li>
	{/each}
</ul>
