<script lang="ts">
	// Request limits: how many movies and how many seasons of shows in how many days. An empty field means "use the
	// default"; 0 means no limit.
	import { t } from '$lib/i18n/index.svelte';
	import { Input } from '$lib/components/ui/input';

	export type QuotaValues = { movieLimit: string; movieDays: string; tvLimit: string; tvDays: string };

	let {
		values = $bindable(),
		defaults,
		disabled = false,
	}: {
		values: QuotaValues;
		/** shown as the placeholder: what applies when a field is empty */
		defaults?: { movieLimit: number; movieDays: number; tvLimit: number; tvDays: number } | null;
		disabled?: boolean;
	} = $props();
</script>

<div class="grid gap-3 text-sm">
	<div class="grid grid-cols-[minmax(0,1fr)_6rem_6rem] items-center gap-2">
		<span></span>
		<span class="text-xs text-muted-foreground">{t('users.quota_limit')}</span>
		<span class="text-xs text-muted-foreground">{t('users.quota_days')}</span>

		<span class="font-medium">{t('users.quota_movies')}</span>
		<Input type="number" min="0" max="100000" class="h-8" {disabled} bind:value={values.movieLimit} placeholder={defaults ? String(defaults.movieLimit) : ''} aria-label="{t('users.quota_movies')} · {t('users.quota_limit')}" />
		<Input type="number" min="0" max="3650" class="h-8" {disabled} bind:value={values.movieDays} placeholder={defaults ? String(defaults.movieDays) : ''} aria-label="{t('users.quota_movies')} · {t('users.quota_days')}" />

		<span class="font-medium">{t('users.quota_shows')}</span>
		<Input type="number" min="0" max="100000" class="h-8" {disabled} bind:value={values.tvLimit} placeholder={defaults ? String(defaults.tvLimit) : ''} aria-label="{t('users.quota_shows')} · {t('users.quota_limit')}" />
		<Input type="number" min="0" max="3650" class="h-8" {disabled} bind:value={values.tvDays} placeholder={defaults ? String(defaults.tvDays) : ''} aria-label="{t('users.quota_shows')} · {t('users.quota_days')}" />
	</div>
	<p class="text-xs text-muted-foreground">{t('users.quota_hint')}</p>
</div>
