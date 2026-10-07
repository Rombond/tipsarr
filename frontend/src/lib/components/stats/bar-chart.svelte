<script lang="ts">
	// A one-series bar chart in plain HTML: horizontal bars (names on the left) or columns (names
	// underneath). Marks are thin, rounded only at the data end, and sit on one baseline; the value
	// shows at the mark's tip for the largest bar and in a tooltip for every bar (hover or focus).
	// A visually hidden table carries the same numbers for screen readers.
	type Datum = { label: string; value: number; text: string };

	let {
		data,
		direction = 'horizontal',
		caption,
		labelEvery = 1,
	}: {
		data: Datum[];
		direction?: 'horizontal' | 'vertical';
		/** Names the chart for assistive tech and its hidden table. */
		caption: string;
		/** Columns only: show every n-th label under the bars (hours of the day). */
		labelEvery?: number;
	} = $props();

	const max = $derived(Math.max(0, ...data.map((d) => d.value)));
	const pct = (v: number) => (max > 0 ? Math.max((v / max) * 100, v > 0 ? 1.5 : 0) : 0);
	const top = $derived(data.findIndex((d) => d.value === max && max > 0));
</script>

<figure class="viz" aria-label={caption}>
	{#if direction === 'horizontal'}
		<ul class="grid gap-1.5">
			{#each data as d, i (d.label)}
				<li class="grid grid-cols-[minmax(5rem,9rem)_minmax(0,1fr)_4.5rem] items-center gap-3 text-sm">
					<span class="truncate text-muted-foreground" title={d.label}>{d.label}</span>
					<span class="group relative flex h-5 items-center">
						<button type="button" class="bar-h" style="width:{pct(d.value)}%" aria-label="{d.label}: {d.text}"></button>
						<span class="tip" role="tooltip">{d.label} · {d.text}</span>
					</span>
					<span class="text-xs text-foreground tabular-nums">{i === top ? d.text : ''}</span>
				</li>
			{/each}
		</ul>
	{:else}
		<div class="grid h-44 grid-flow-col auto-cols-fr items-end gap-1 border-b border-border px-1">
			{#each data as d, i (d.label + i)}
				<div class="group relative flex h-full flex-col items-center justify-end">
					<button type="button" class="bar-v" style="height:{pct(d.value)}%" aria-label="{d.label}: {d.text}"></button>
					{#if i === top}<span class="absolute -top-0.5 -translate-y-full text-[11px] whitespace-nowrap text-foreground tabular-nums">{d.text}</span>{/if}
					<span class="tip" role="tooltip">{d.label} · {d.text}</span>
				</div>
			{/each}
		</div>
		<div class="mt-1 grid grid-flow-col auto-cols-fr gap-1 px-1 text-center text-[11px] text-muted-foreground">
			{#each data as d, i (d.label + i)}
				<span class="truncate">{i % labelEvery === 0 ? d.label : ''}</span>
			{/each}
		</div>
	{/if}
	<table class="sr-only">
		<caption>{caption}</caption>
		<tbody>
			{#each data as d (d.label)}
				<tr><th scope="row">{d.label}</th><td>{d.text}</td></tr>
			{/each}
		</tbody>
	</table>
</figure>

<style>
	.viz {
		--bar: #2a78d6;
		--surface: var(--background);
	}
	:global(.dark) .viz {
		--bar: #3987e5;
	}
	.bar-h,
	.bar-v {
		background: var(--bar);
		border: 0;
		padding: 0;
		cursor: default;
		transition: opacity 120ms;
	}
	.bar-h {
		height: 14px;
		max-width: 100%;
		border-radius: 0 4px 4px 0; /* rounded at the data end only, square on the baseline */
	}
	.bar-v {
		width: 100%;
		max-width: 24px;
		min-height: 0;
		border-radius: 4px 4px 0 0;
	}
	.bar-h:hover,
	.bar-v:hover,
	.bar-h:focus-visible,
	.bar-v:focus-visible {
		opacity: 0.8;
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
	.tip {
		pointer-events: none;
		position: absolute;
		z-index: 10;
		bottom: calc(100% + 4px);
		left: 50%;
		transform: translateX(-50%);
		white-space: nowrap;
		border-radius: 6px;
		background: var(--popover);
		color: var(--popover-foreground);
		box-shadow: 0 0 0 1px var(--border), 0 4px 12px rgb(0 0 0 / 0.25);
		padding: 2px 8px;
		font-size: 12px;
		opacity: 0;
		transition: opacity 100ms;
	}
	.group:hover > .tip,
	.group:focus-within > .tip {
		opacity: 1;
	}
</style>
