<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import TimerIcon from '@lucide/svelte/icons/timer';

	// The demo gateway tells when this sandbox is deleted; without a gateway (local try) there is no timer.
	let endsAt: number | null = $state(null);
	let now = $state(Date.now());

	$effect(() => {
		if (!auth.demo) return;
		fetch('/__demo/session', { credentials: 'same-origin' })
			.then((r) => (r.ok ? r.json() : null))
			.then((s: { endsAt?: number } | null) => {
				if (s?.endsAt) endsAt = s.endsAt * 1000;
			})
			.catch(() => {});
		const id = setInterval(() => (now = Date.now()), 30_000);
		return () => clearInterval(id);
	});

	const minutes = $derived(endsAt === null ? null : Math.max(0, Math.ceil((endsAt - now) / 60_000)));

	// the sandbox is gone: a reload hands out a fresh one
	$effect(() => {
		if (endsAt !== null && now >= endsAt + 5_000) location.reload();
	});
</script>

{#if auth.demo}
	<div
		class="flex items-center justify-center gap-1.5 border-b border-sky-500/30 bg-sky-500/10 px-3 py-1 text-center text-[11px] text-sky-700 dark:text-sky-300"
		role="status"
	>
		<TimerIcon class="size-3.5 shrink-0" />
		<span>
			<strong>{t('banner.demo')}</strong>
			{#if minutes !== null}
				· {t('banner.demo_left', { minutes })}
			{/if}
		</span>
	</div>
{/if}
