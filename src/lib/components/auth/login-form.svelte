<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';

	let username = $state('');
	let password = $state('');
	let error: string | null = $state(null);
	let submitting = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		submitting = true;
		try {
			await auth.login(username, password);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			submitting = false;
		}
	}
</script>

<div class="flex min-h-svh items-center justify-center px-4">
	<Card class="w-full max-w-sm">
		<CardHeader>
			<CardTitle>Sign in to TipsArr</CardTitle>
			<CardDescription>Use your Jellyfin account credentials.</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="grid gap-3" onsubmit={handleSubmit}>
				<Input placeholder="Username" bind:value={username} autocomplete="username" required />
				<Input placeholder="Password" type="password" bind:value={password} autocomplete="current-password" required />
				{#if error}
					<p class="text-sm text-destructive">{error}</p>
				{/if}
				<Button type="submit" disabled={submitting}>
					{submitting ? 'Signing in…' : 'Sign in'}
				</Button>
			</form>
		</CardContent>
	</Card>
</div>
