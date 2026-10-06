<script lang="ts">
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import CheckIcon from '@lucide/svelte/icons/circle-check';
	import CircleIcon from '@lucide/svelte/icons/circle';
	import XIcon from '@lucide/svelte/icons/x';

	const KEY = 'tipsarr-checklist-dismissed';

	let settings = $state<Schemas['SettingsBody'] | null>(null);
	let sync = $state<Schemas['SyncStatusBody'] | null>(null);
	let instances = $state<Schemas['InstanceView'][]>([]);
	let dismissed = $state(false);
	let ready = $state(false);

	$effect(() => {
		try {
			dismissed = localStorage.getItem(KEY) === '1';
		} catch {
			/* ignore */
		}
		if (!auth.isAdmin || dismissed) return;
		Promise.all([unwrap(api.GET('/admin/settings')), unwrap(api.GET('/admin/sync')), unwrap(api.GET('/admin/servarr'))])
			.then(([s, y, i]) => {
				settings = s;
				sync = y;
				instances = i;
				ready = true;
			})
			.catch(() => {});
	});

	const steps = $derived(
		settings && sync
			? [
					{ done: settings.tmdbConfigured, label: 'Add your TMDB key', hint: 'Needed for everything you browse', href: '/admin/settings' },
					{ done: settings.jellyfinApiKeyConfigured, label: 'Add a Jellyfin API key', hint: 'Lets Tipsarr see your library and watch history', href: '/admin/settings' },
					{ done: sync.movies + sync.shows > 0, label: 'Sync your library', hint: 'Shows what you already have on every poster', href: '/admin/settings' },
					{ done: instances.some((i) => i.kind === 'radarr'), label: 'Connect Radarr', hint: 'Where approved movies go', href: '/admin/services' },
					{ done: instances.some((i) => i.kind === 'sonarr'), label: 'Connect Sonarr', hint: 'Where approved shows go', href: '/admin/services' },
				]
			: [],
	);
	const remaining = $derived(steps.filter((s) => !s.done).length);

	function dismiss() {
		dismissed = true;
		try {
			localStorage.setItem(KEY, '1');
		} catch {
			/* ignore */
		}
	}
</script>

{#if auth.isAdmin && ready && !dismissed && remaining > 0}
	<section class="rounded-xl border border-border bg-card p-4">
		<div class="mb-2 flex items-start justify-between gap-2">
			<div>
				<h2 class="font-semibold">Finish setting up Tipsarr</h2>
				<p class="text-xs text-muted-foreground">{remaining} step{remaining === 1 ? '' : 's'} left. {auth.dryRun ? 'Dry-run is on, so nothing can be sent to Radarr or Sonarr yet.' : ''}</p>
			</div>
			<button type="button" class="cursor-pointer text-muted-foreground hover:text-foreground" aria-label="Hide this checklist" onclick={dismiss}>
				<XIcon class="size-4" />
			</button>
		</div>
		<ul class="grid gap-1 text-sm sm:grid-cols-2">
			{#each steps as s (s.label)}
				<li>
					<a href={s.href} class="flex items-start gap-2 rounded-md p-1.5 hover:bg-accent {s.done ? 'text-muted-foreground' : ''}">
						{#if s.done}<CheckIcon class="mt-0.5 size-4 shrink-0 text-emerald-500" />{:else}<CircleIcon class="mt-0.5 size-4 shrink-0" />{/if}
						<span>
							<span class="block {s.done ? 'line-through' : 'font-medium'}">{s.label}</span>
							{#if !s.done}<span class="block text-xs text-muted-foreground">{s.hint}</span>{/if}
						</span>
					</a>
				</li>
			{/each}
		</ul>
	</section>
{/if}
