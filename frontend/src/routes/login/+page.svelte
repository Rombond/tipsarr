<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import LoginForm from '$lib/components/auth/login-form.svelte';
	import SetupForm from '$lib/components/auth/setup-form.svelte';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import { t, i18n, LOCALES, type Locale } from '$lib/i18n/index.svelte';
</script>

<div class="flex min-h-svh flex-col items-center justify-center gap-6 bg-[radial-gradient(ellipse_at_top,var(--accent),transparent_60%)] px-4">
	<div class="flex items-center gap-2 font-bold text-2xl">
		<span class="inline-flex size-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">T</span>
		Tipsarr
	</div>
	{#if auth.configured === false}
		<SetupForm />
	{:else}
		<LoginForm />
	{/if}
	<p class="text-xs text-muted-foreground">{t('brand.tagline')}</p>
	<SimpleSelect
		label={t('lang.title')}
		value={i18n.locale}
		options={LOCALES.map((l) => ({ value: l.code, label: l.name }))}
		onchange={(v) => i18n.set(v as Locale)}
		class="h-8 min-w-32 text-xs"
	/>
</div>
