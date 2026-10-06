// Pending-request and open-issue counters for the navigation badge (admins), kept fresh by SSE.
import { api, unwrap } from '$lib/api/client';
import { onEvent, stream } from '$lib/events.svelte';

class Counts {
	pending = $state(0);
	issues = $state(0); // open issues

	async refresh() {
		try {
			const [r, i] = await Promise.all([unwrap(api.GET('/requests/counts')), unwrap(api.GET('/issues/counts'))]);
			this.pending = r.pending;
			this.issues = i.open;
		} catch {
			/* not critical */
		}
	}

	/** Start tracking; returns a stop function. Call from an $effect. */
	track(): () => void {
		this.refresh();
		void stream.reconnects;
		const offRequests = onEvent('request.updated', () => this.refresh());
		const offIssues = onEvent('issue.updated', () => this.refresh());
		return () => {
			offRequests();
			offIssues();
		};
	}
}

export const counts = new Counts();
