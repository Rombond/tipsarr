<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { goto } from '$app/navigation';
	import { fmtDuration } from '$lib/i18n/format';
	import type { Schemas } from '$lib/api/client';
	import CheckIcon from '@lucide/svelte/icons/check';
	import StarIcon from '@lucide/svelte/icons/star';

	let { item }: { item: Schemas['LibraryItem'] } = $props();

	const meta = $derived(
		[item.year || '', item.runtimeMinutes ? fmtDuration(item.runtimeMinutes * 60, t) : ''].filter(Boolean).join(' · '),
	);
</script>

<div class="min-w-0">
	<button
		type="button"
		class="group relative block aspect-[2/3] w-full cursor-pointer overflow-hidden rounded-lg bg-muted text-left shadow-sm ring-1 ring-border/50 transition hover:shadow-lg hover:ring-border focus-visible:ring-2 focus-visible:ring-ring"
		aria-label={item.title}
		onclick={() => goto(`/media/${item.type}/${item.tmdbId}`)}
	>
		{#if item.posterUrl}
			<img src={item.posterUrl} alt="" class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]" loading="lazy" />
		{:else}
			<div class="flex h-full w-full items-center justify-center p-2 text-center text-xs text-muted-foreground">{item.title}</div>
		{/if}
		{#if item.watched}
			<span class="absolute top-1.5 right-1.5 flex size-5 items-center justify-center rounded-full bg-emerald-500 text-white shadow" title={t('library.watched_by_me')}>
				<CheckIcon class="size-3.5" />
				<span class="sr-only">{t('library.watched_by_me')}</span>
			</span>
		{/if}
		{#if item.rating}
			<span class="absolute bottom-1.5 left-1.5 flex items-center gap-0.5 rounded-full bg-black/70 px-1.5 py-0.5 text-[11px] font-medium text-white">
				<StarIcon class="size-3 fill-amber-400 text-amber-400" />{item.rating.toFixed(1)}
			</span>
		{/if}
	</button>
	<p class="mt-1.5 truncate text-sm font-medium" title={item.title}>{item.title}</p>
	{#if meta}<p class="truncate text-xs text-muted-foreground">{meta}</p>{/if}
</div>
