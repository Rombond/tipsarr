<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, unwrap, imageUrl, type Schemas } from '$lib/api/client';
	import SearchIcon from '@lucide/svelte/icons/search';
	import FilmIcon from '@lucide/svelte/icons/film';
	import TvIcon from '@lucide/svelte/icons/tv';
	import UserIcon from '@lucide/svelte/icons/user';

	type Hit = { href: string; title: string; sub: string; img: string | null; kind: 'movie' | 'tv' | 'person' };

	let query = $state('');
	let hits = $state<Hit[]>([]);
	let open = $state(false);
	let active = $state(-1);
	let loading = $state(false);
	let input: HTMLInputElement | undefined = $state();
	let timer: ReturnType<typeof setTimeout> | undefined;
	let seq = 0;

	function toHits(r: Schemas['SearchResult']): Hit[] {
		const titles: Hit[] = r.items.slice(0, 6).map((i) => ({
			href: `/media/${i.type}/${i.tmdbId}`,
			title: i.title,
			sub: `${i.type === 'tv' ? 'TV show' : 'Movie'}${i.releaseDate ? ' · ' + i.releaseDate.slice(0, 4) : ''}`,
			img: imageUrl(i.posterPath, 'w92'),
			kind: i.type,
		}));
		const people: Hit[] = r.people.slice(0, 2).map((p) => ({
			href: `/person/${p.id}`,
			title: p.name,
			sub: p.department || 'Person',
			img: imageUrl(p.profilePath, 'w92'),
			kind: 'person',
		}));
		return [...titles, ...people];
	}

	function onInput() {
		clearTimeout(timer);
		active = -1;
		const q = query.trim();
		if (q.length < 2) {
			hits = [];
			open = false;
			return;
		}
		timer = setTimeout(async () => {
			const mine = ++seq;
			loading = true;
			try {
				const r = await unwrap(api.GET('/search', { params: { query: { q } } }));
				if (mine !== seq) return; // a newer keystroke superseded this answer
				hits = toHits(r);
				open = true;
			} catch {
				hits = [];
			} finally {
				if (mine === seq) loading = false;
			}
		}, 250);
	}

	function go(href: string) {
		open = false;
		query = '';
		hits = [];
		input?.blur();
		goto(href);
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowDown' && hits.length) {
			e.preventDefault();
			open = true;
			active = (active + 1) % hits.length;
		} else if (e.key === 'ArrowUp' && hits.length) {
			e.preventDefault();
			active = (active - 1 + hits.length) % hits.length;
		} else if (e.key === 'Enter') {
			if (open && active >= 0 && hits[active]) go(hits[active].href);
			else if (query.trim()) {
				open = false;
				goto(`/search?q=${encodeURIComponent(query.trim())}`);
			}
		} else if (e.key === 'Escape') {
			open = false;
			input?.blur();
		}
	}

	// "/" focuses the search from anywhere (unless typing in a field)
	function globalKey(e: KeyboardEvent) {
		const t = e.target as HTMLElement | null;
		const typing = t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT' || t.isContentEditable);
		if (e.key === '/' && !typing && !e.metaKey && !e.ctrlKey) {
			e.preventDefault();
			input?.focus();
		}
	}
</script>

<svelte:window onkeydown={globalKey} />

<div class="relative w-full">
	<SearchIcon class="pointer-events-none absolute top-1/2 left-4 z-10 size-5 -translate-y-1/2 text-muted-foreground" />
	<input
		bind:this={input}
		bind:value={query}
		oninput={onInput}
		onkeydown={onKeydown}
		onfocus={() => hits.length && (open = true)}
		onblur={() => setTimeout(() => (open = false), 150)}
		type="search"
		role="combobox"
		aria-expanded={open}
		aria-controls="search-hits"
		aria-label="Search movies, TV shows and people"
		placeholder="Search movies, TV shows, people…  ( / )"
		autocomplete="off"
		class="h-11 w-full rounded-full border border-border/60 bg-background/60 pr-4 pl-11 text-sm shadow-sm outline-none backdrop-blur-md placeholder:text-muted-foreground focus-visible:border-ring focus-visible:bg-background/90 focus-visible:ring-[3px] focus-visible:ring-ring/40 [&::-webkit-search-cancel-button]:hidden"
	/>
	{#if open}
		<ul
			id="search-hits"
			role="listbox"
			class="absolute top-full right-0 left-0 z-50 mt-1 max-h-[70vh] overflow-y-auto rounded-lg border border-border bg-popover p-1 shadow-lg"
		>
			{#each hits as h, i (h.href)}
				<li role="option" aria-selected={i === active}>
					<a
						href={h.href}
						class="flex cursor-pointer items-center gap-3 rounded-md p-2 text-sm hover:bg-accent {i === active ? 'bg-accent' : ''}"
						onmousedown={(e) => {
							e.preventDefault();
							go(h.href);
						}}
					>
						{#if h.img}
							<img src={h.img} alt="" class="h-12 w-8 shrink-0 rounded object-cover" loading="lazy" />
						{:else}
							<span class="flex h-12 w-8 shrink-0 items-center justify-center rounded bg-muted text-muted-foreground">
								{#if h.kind === 'person'}<UserIcon class="size-4" />{:else if h.kind === 'tv'}<TvIcon class="size-4" />{:else}<FilmIcon class="size-4" />{/if}
							</span>
						{/if}
						<span class="min-w-0">
							<span class="block truncate font-medium">{h.title}</span>
							<span class="block truncate text-xs text-muted-foreground">{h.sub}</span>
						</span>
					</a>
				</li>
			{:else}
				{#if !loading}<li class="p-3 text-sm text-muted-foreground">No quick matches. Press Enter to search.</li>{/if}
			{/each}
			{#if hits.length}
				<li>
					<a
						href="/search?q={encodeURIComponent(query.trim())}"
						class="block cursor-pointer rounded-md p-2 text-center text-xs text-muted-foreground hover:bg-accent"
						onmousedown={(e) => {
							e.preventDefault();
							go(`/search?q=${encodeURIComponent(query.trim())}`);
						}}>See all results</a
					>
				</li>
			{/if}
		</ul>
	{/if}
</div>
