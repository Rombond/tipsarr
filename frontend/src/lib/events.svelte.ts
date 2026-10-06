// Live updates over Server-Sent Events. Events are a "something changed" nudge: components
// refetch over REST, and always after a reconnect (anything may have been missed).
//
// Browsers allow only ~6 HTTP/1.1 connections per host, and every SSE stream holds one for as
// long as the tab lives. So all tabs share ONE stream: a Web Lock elects a leader tab that owns
// the EventSource and relays every event to the other tabs over a BroadcastChannel. When the
// leader tab closes, the lock passes to another tab, which reconnects (and everyone refetches).

type Handler = (data: any) => void;

const TYPES = ['request.updated', 'request.progress', 'media.available', 'sync.status', 'suggestions.updated'];
const LOCK = 'tipsarr-sse';

const handlers = new Map<string, Set<Handler>>();
let source: EventSource | null = null;
let channel: BroadcastChannel | null = null;
let abort: AbortController | null = null;
let releaseLock: (() => void) | null = null;
let started = false;
let everLeader = false;

export const stream = $state({ connected: false, reconnects: 0 });

function dispatch(type: string, data: unknown) {
	handlers.get(type)?.forEach((h) => h(data));
}

function openSource() {
	closeSource();
	const es = new EventSource('/api/v1/events');
	source = es;
	es.onopen = () => {
		stream.connected = true;
		if (everLeader) {
			stream.reconnects++; // consumers refetch when this changes
			channel?.postMessage({ kind: 'reconnect' });
		}
		everLeader = true;
	};
	es.onerror = () => {
		stream.connected = false; // the browser reconnects by itself
	};
	for (const type of TYPES) {
		es.addEventListener(type, (e) => {
			let data: unknown = null;
			try {
				data = JSON.parse((e as MessageEvent).data);
			} catch {
				return;
			}
			dispatch(type, data);
			channel?.postMessage({ kind: 'event', type, data }); // relay to follower tabs
		});
	}
}

function closeSource() {
	source?.close();
	source = null;
}

export function connectEvents() {
	if (started || typeof EventSource === 'undefined') return;
	started = true;

	if (typeof BroadcastChannel !== 'undefined') {
		channel = new BroadcastChannel('tipsarr-events');
		channel.onmessage = (m) => {
			if (m.data?.kind === 'event') dispatch(m.data.type, m.data.data);
			else if (m.data?.kind === 'reconnect') stream.reconnects++;
		};
	}

	if (typeof navigator !== 'undefined' && navigator.locks && channel) {
		// followers wait here; the leader holds the lock (and the connection) until its tab closes
		abort = new AbortController();
		stream.connected = true; // followers receive relayed events
		navigator.locks
			.request(LOCK, { signal: abort.signal }, () => {
				openSource();
				return new Promise<void>((resolve) => (releaseLock = resolve));
			})
			.catch(() => {}); // aborted on logout
	} else {
		openSource(); // no Web Locks / BroadcastChannel: one stream per tab
	}
}

export function disconnectEvents() {
	abort?.abort();
	abort = null;
	releaseLock?.();
	releaseLock = null;
	closeSource();
	channel?.close();
	channel = null;
	started = false;
	everLeader = false;
	stream.connected = false;
}

/** Subscribe to an event type; returns the unsubscribe function. */
export function onEvent(type: string, handler: Handler): () => void {
	let set = handlers.get(type);
	if (!set) handlers.set(type, (set = new Set()));
	set.add(handler);
	return () => set.delete(handler);
}
