<script lang="ts">
	import { page } from '$app/state';
	import { Button } from '$lib/components/ui/button';
</script>

<svelte:head><title>{page.status} · Tipsarr</title></svelte:head>

<div class="grid min-h-[60vh] place-content-center justify-items-center gap-3 text-center">
	<p class="font-black text-6xl text-muted-foreground/40">{page.status}</p>
	<h1 class="font-semibold text-xl">{page.status === 404 ? 'Nothing here' : 'Something went wrong'}</h1>
	<p class="max-w-sm text-sm text-muted-foreground">
		{page.status === 404 ? 'That page does not exist, or it moved.' : (page.error?.message ?? 'Please try again.')}
	</p>
	<div class="flex gap-2">
		<Button onclick={() => history.back()} variant="outline">Go back</Button>
		<Button href="/discover">Discover</Button>
	</div>
</div>
