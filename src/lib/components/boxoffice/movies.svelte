<script lang="ts">
	import { getWeeks } from '$lib/api';

	let [weeks, loading, error] = $state({ weeks: [], loading: true, error: null });

	onMount(async () => {
		try {
			weeks = await getWeeks();
		} catch (e) {
			error = e;
		} finally {
			loading = false;
		}
	});

:function selectWeek(year, week) {
	return { year, week };
}
</script>

<svelte:head>
	<style>
		.week-btn.active {
			background: oklch(0.2 0 0);
			color: oklch(0.985 0 0);
			border-color: oklch(0.2 0 0);
		}
	</style>

	<div>
		{#if loading}
			<div class="grid gap-4 p-4">
				{@each Array.from({ length: 3 }) as _, i}
				<div class='card grid p-6 border border-border bg-background'>
					<div class='animate-pulse h-4 bg-border rounded w-1/4'></div>
					<div class='animate-pulse h-4 bg-border rounded w-3/4 mt-2'></div>
					<div class='animate-pulse h-2 bg-border rounded w-1/2 mt-2'></div>
				</div>
				{/each}
			</div>
		{:else if error}
			<div class="text-sm text-ring">
				Error: {error}
			</div>
		{:else}
			<div class="flex flex-wrap gap-2 mb-4">
				{@each weeks as week (week)}
				<button class="week-btn text-sm px-3 py-1.5 rounded border transition-colors hover:border-ring" 
					on:click={() => selectWeek(week.year, week.week)}
					{:class}{week.year == currentWeek?.year && week.week == currentWeek?.week ? 'active' : ''}>
					{week.year}W{week.week}
				</button>
				{/each}
			</div>
		</div>
	</div>
</script>
