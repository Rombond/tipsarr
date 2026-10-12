<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { fmtDate } from '$lib/i18n/format';
	import { api, unwrap, imageUrl, errorText, type Schemas } from '$lib/api/client';
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
			error = errorText(e);
		} finally {
			loading = null;
		}
	}

	const date = (d?: string) => fmtDate(d);
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
				<span class="font-medium">{s.name || t('seasons.season', { n: s.number })}</span>
				<span class="flex items-center gap-2 text-muted-foreground">
					{#if s.episodeCount > 0 && (s.libraryEpisodes ?? 0) >= s.episodeCount}
						<span class="text-emerald-600 dark:text-emerald-400">{t('req.season_in_library')}</span> ·
					{:else if (s.libraryEpisodes ?? 0) > 0}
						<span class="text-teal-600 dark:text-teal-400">{t('req.season_partial', { have: s.libraryEpisodes ?? 0, total: s.episodeCount })}</span>
					{/if}
					{#if s.monitored && !(s.episodeCount > 0 && (s.libraryEpisodes ?? 0) >= s.episodeCount)}
						<span class="text-sky-600 dark:text-sky-400">{t('req.season_monitored')}</span> ·
					{/if}
					{t('seasons.episodes', { count: s.episodeCount })}{#if s.airDate}&nbsp;· {s.airDate.slice(0, 4)}{/if}
					<ChevronIcon class="size-4 transition-transform {openSeason === s.number ? 'rotate-180' : ''}" />
				</span>
			</button>
			{#if openSeason === s.number}
				<div class="border-t border-border bg-muted/30 px-3 py-2">
					{#if loading === s.number}
						<p class="py-2 text-muted-foreground">{t('seasons.loading')}</p>
					{:else if error}
						<p class="py-2 text-destructive">{error}</p>
					{:else}
						<ol class="grid gap-3">
							{#each episodes[s.number] ?? [] as e (e.number)}
								{@const still = imageUrl(e.stillPath, 'w300')}
								<li class="flex gap-3">
									<div class="relative aspect-video w-36 shrink-0 overflow-hidden rounded-md bg-muted sm:w-48">
										{#if still}
											<img src={still} alt="" class="h-full w-full object-cover" loading="lazy" />
										{:else}
											<div class="flex h-full w-full items-center justify-center text-xs text-muted-foreground">{t('seasons.no_preview')}</div>
										{/if}
										<span class="absolute bottom-1 left-1 rounded bg-black/70 px-1.5 py-0.5 text-[11px] font-semibold text-white">E{e.number}</span>
									</div>
									<div class="min-w-0 py-0.5">
										<p class="font-medium leading-snug">
											{e.name}
											{#if e.state}
												<span class="ml-1 rounded px-1.5 py-0.5 align-middle text-[11px] font-semibold {e.state === 'available' ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400' : e.state === 'monitored' ? 'bg-sky-500/15 text-sky-600 dark:text-sky-400' : 'bg-muted text-muted-foreground'}">{t(e.state === 'available' ? 'seasons.ep_available' : e.state === 'monitored' ? 'seasons.ep_monitored' : 'seasons.ep_missing')}</span>
											{/if}
										</p>
										<p class="text-xs text-muted-foreground">
											{date(e.airDate)}{#if e.runtimeMinutes} · {t('time.min', { count: e.runtimeMinutes })}{/if}{#if e.voteAverage} · ★ {e.voteAverage.toFixed(1)}{/if}
										</p>
										{#if e.overview}<p class="mt-1 line-clamp-3 text-xs text-muted-foreground">{e.overview}</p>{/if}
									</div>
								</li>
							{/each}
						</ol>
					{/if}
				</div>
			{/if}
		</li>
	{/each}
</ul>
