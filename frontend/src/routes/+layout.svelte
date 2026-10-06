<script lang="ts">
	import './layout.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import AppShell from '$lib/components/layout/app-shell.svelte';
	import { connectEvents, disconnectEvents } from '$lib/events.svelte';
	import { theme } from '$lib/theme.svelte';
	import { i18n } from '$lib/i18n/index.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import Toaster from '$lib/components/ui/toaster/toaster.svelte';

	let { children } = $props();

	$effect(() => {
		theme.init();
		i18n.init();
	});

	// the account's language (profile) wins once we know who is signed in
	$effect(() => {
		i18n.useProfileLanguage(auth.user?.language);
	});

	$effect(() => {
		auth.bootstrap();
	});

	$effect(() => {
		if (auth.status === 'authenticated') connectEvents();
		else disconnectEvents();
	});

	$effect(() => {
		const isLoginRoute = page.url.pathname === '/login';
		if (auth.status === 'unauthenticated' && !isLoginRoute) {
			goto('/login');
		} else if (auth.status === 'authenticated' && isLoginRoute) {
			goto('/discover');
		}
	});
</script>


{#if page.url.pathname === '/login'}
	{@render children()}
{:else if auth.status === 'authenticated'}
	<AppShell>
		{@render children()}
	</AppShell>
{:else}
	<div class="flex min-h-svh items-center justify-center text-muted-foreground text-sm">
		{auth.status === 'unauthenticated' ? t('loading.redirect') : t('common.loading')}
	</div>
{/if}

<Toaster />
