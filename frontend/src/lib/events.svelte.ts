// Live updates over Server-Sent Events. Events are a "something changed" nudge: components
// refetch over REST, and always after a reconnect (anything may have been missed).

type Handler = (data: any) => void;

const handlers = new Map<string, Set<Handler>>();
let source: EventSource | null = null;
let everConnected = false;

export const stream = $state({ connected: false, reconnects: 0 });

export function connectEvents() {
	if (source || typeof EventSource === 'undefined') return;
	source = new EventSource('/api/v1/events');
	source.onopen = () => {
		stream.connected = true;
		if (everConnected) stream.reconnects++; // consumers refetch when this changes
		everConnected = true;
	};
	source.onerror = () => {
		stream.connected = false; // the browser reconnects by itself
	};
	for (const type of ['request.updated', 'request.progress', 'media.available', 'sync.status', 'suggestions.updated']) {
		source.addEventListener(type, (e) => {
			let data: unknown = null;
			try {
				data = JSON.parse((e as MessageEvent).data);
			} catch {
				return;
			}
			handlers.get(type)?.forEach((h) => h(data));
		});
	}
}

export function disconnectEvents() {
	source?.close();
	source = null;
	everConnected = false;
	stream.connected = false;
}

/** Subscribe to an event type; returns the unsubscribe function. */
export function onEvent(type: string, handler: Handler): () => void {
	let set = handlers.get(type);
	if (!set) handlers.set(type, (set = new Set()));
	set.add(handler);
	return () => set.delete(handler);
}
