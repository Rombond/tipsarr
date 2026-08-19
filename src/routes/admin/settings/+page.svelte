<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';

	onMount(() => {
		if (!auth.isAdmin) goto('/discover');
	});

	const envInfo = [
		{ label: 'BoxArr URL', value: import.meta.env.VITE_BOXARR_URL },
		{ label: 'SuggestArr URL', value: import.meta.env.VITE_SUGGESTARR_URL },
		{ label: 'Seerr URL', value: import.meta.env.VITE_SEERR_URL },
		{ label: 'Radarr URL', value: import.meta.env.VITE_RADARR_URL },
	];
</script>

<svelte:head>
	<meta name="description" content="TipsArr Settings" />
</svelte:head>

<div class="grid gap-4">
	<h1 class="font-bold text-2xl">Settings</h1>
	<Card>
		<CardHeader>
			<CardTitle>Backend configuration</CardTitle>
			<CardDescription>Configured via environment variables at build time.</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-1.5 text-sm">
			{#each envInfo as info (info.label)}
				<div class="flex justify-between border-b border-border py-1.5">
					<span class="text-muted-foreground">{info.label}</span>
					<span class="font-mono">{info.value || '(not set)'}</span>
				</div>
			{/each}
		</CardContent>
	</Card>
</div>
