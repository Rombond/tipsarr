<script lang="ts">
	import './layout.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import AppShell from '$lib/components/layout/app-shell.svelte';
	import { connectEvents, disconnectEvents } from '$lib/events.svelte';
	import { theme } from '$lib/theme.svelte';
	import Toaster from '$lib/components/ui/toaster/toaster.svelte';

	let { children } = $props();

	$effect(() => {
		theme.init();
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
		{auth.status === 'unauthenticated' ? 'Redirecting to login…' : 'Loading…'}
	</div>
{/if}

<Toaster />
