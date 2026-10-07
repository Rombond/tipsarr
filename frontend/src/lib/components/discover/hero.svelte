<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, imageUrl, type MediaItem } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import RequestFlow from '$lib/components/requests/request-flow.svelte';
	import StarIcon from '@lucide/svelte/icons/star';

	let slides = $state<MediaItem[]>([]);
	let index = $state(0);
	let loading = $state(true);
	let paused = $state(false);
	let requested = $state<Record<string, string>>({});

	$effect(() => {
		unwrap(api.GET('/discover/trending', { params: { query: { page: 1 } } }))
			.then((r) => (slides = r.items.filter((i) => i.backdropPath).slice(0, 6)))
			.catch(() => {})
			.finally(() => (loading = false));
	});

	$effect(() => {
		if (slides.length < 2 || paused || matchMedia('(prefers-reduced-motion: reduce)').matches) return;
		const t = setInterval(() => (index = (index + 1) % slides.length), 8000);
		return () => clearInterval(t);
	});

	const current = $derived(slides[index]);
	const key = (i: MediaItem) => `${i.type}:${i.tmdbId}`;
	const status = (i: MediaItem) => requested[key(i)] ?? i.requestStatus ?? null;

	let flow: { start: () => void } | undefined = $state();
	let flowBusy = $state(false);
	function request(i: MediaItem) {
		flow?.start(); // the same request dialog as everywhere (seasons of a show, quality profile...)
	}
</script>

{#if current}
	<RequestFlow bind:this={flow} bind:busy={flowBusy} type={current.type} tmdbId={current.tmdbId} title={current.title} onrequested={(r) => (requested = { ...requested, [key(current)]: r.status })} />
{/if}

{#if loading}
	<Skeleton class="h-56 w-full rounded-xl md:h-80" />
{:else if current}
	<section
		class="relative overflow-hidden rounded-xl bg-muted"
		onmouseenter={() => (paused = true)}
		onmouseleave={() => (paused = false)}
		aria-roledescription="carousel"
		aria-label={t('hero.aria')}
	>
		{#key key(current)}
			<img
				src={imageUrl(current.backdropPath, 'w1280')}
				alt=""
				class="h-56 w-full animate-in fade-in object-cover duration-700 md:h-80"
			/>
		{/key}
		<div class="absolute inset-0 bg-gradient-to-t from-black/85 via-black/40 to-transparent md:bg-gradient-to-r md:from-black/80 md:via-black/30"></div>
		<div class="absolute inset-x-0 bottom-0 grid max-w-2xl gap-2 p-4 text-white md:p-8">
			<h2 class="font-bold text-2xl drop-shadow md:text-4xl">{current.title}</h2>
			<p class="flex items-center gap-2 text-sm text-white/80">
				{#if current.type === 'tv'}<span class="rounded bg-white/20 px-1.5 py-px text-xs">{t('type.tv_short')}</span>{/if}
				{#if current.releaseDate}<span>{current.releaseDate.slice(0, 4)}</span>{/if}
				{#if current.voteAverage}<span class="inline-flex items-center gap-0.5"><StarIcon class="size-3.5 fill-amber-400 text-amber-400" />{current.voteAverage.toFixed(1)}</span>{/if}
				{#if current.availability === 'available'}<span class="rounded bg-emerald-500/80 px-1.5 py-px text-xs">{t('media.available')}</span>{/if}
			</p>
			{#if current.overview}<p class="line-clamp-2 hidden text-sm text-white/80 sm:block md:line-clamp-3">{current.overview}</p>{/if}
			<div class="flex gap-2 pt-1">
				<Button size="sm" onclick={() => goto(`/media/${current.type}/${current.tmdbId}`)}>{t('common.details')}</Button>
				{#if current.availability !== 'available' && !status(current)}
					<Button size="sm" variant="secondary" onclick={() => request(current)}>{t('media.request')}</Button>
				{:else if status(current)}
					<Button size="sm" variant="secondary" disabled>{status(current) === 'pending' ? t('media.requested') : t('media.approved')}</Button>
				{/if}
			</div>
		</div>
		{#if slides.length > 1}
			<div class="absolute right-3 bottom-3 flex gap-1.5 md:right-6 md:bottom-5">
				{#each slides as s, i (key(s))}
					<button
						type="button"
						class="h-1.5 cursor-pointer rounded-full transition-all {i === index ? 'w-6 bg-white' : 'w-1.5 bg-white/50 hover:bg-white/80'}"
						aria-label={t('hero.show', { title: s.title })}
						onclick={() => (index = i)}
					></button>
				{/each}
			</div>
		{/if}
	</section>
{/if}
