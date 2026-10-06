<script lang="ts">
	// Horizontal scroller with arrow buttons (pointer devices) and edge-to-edge touch scrolling.
	import type { Snippet } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';

	let { title, children, onscrollend }: { title?: string; children: Snippet; onscrollend?: () => void } = $props();

	let el: HTMLDivElement | undefined = $state();

	function by(dir: 1 | -1) {
		el?.scrollBy({ left: dir * el.clientWidth * 0.85, behavior: 'smooth' });
	}

	function onScroll() {
		if (el && el.scrollLeft + el.clientWidth >= el.scrollWidth - 400) onscrollend?.();
	}
</script>

<section class="grid gap-2">
	{#if title}
		<div class="flex items-center justify-between">
			<h2 class="font-semibold text-lg">{title}</h2>
			<div class="flex gap-1 [@media(hover:none)]:hidden">
				<Button variant="outline" size="icon-sm" aria-label="Scroll left" onclick={() => by(-1)}><ChevronLeftIcon /></Button>
				<Button variant="outline" size="icon-sm" aria-label="Scroll right" onclick={() => by(1)}><ChevronRightIcon /></Button>
			</div>
		</div>
	{/if}
	<div
		bind:this={el}
		class="no-scrollbar -mx-4 flex snap-x scroll-px-4 gap-3 overflow-x-auto px-4 pb-2 md:-mx-8 md:scroll-px-8 md:px-8"
		onscroll={onScroll}
	>
		{@render children()}
	</div>
</section>

<style>
	.no-scrollbar {
		scrollbar-width: none;
	}
	.no-scrollbar::-webkit-scrollbar {
		display: none;
	}
</style>
