// Tiny toast queue: success / error messages that fade out by themselves.
export type Toast = { id: number; kind: 'success' | 'error' | 'info'; text: string };

let next = 1;

class ToastState {
	items = $state<Toast[]>([]);

	push(kind: Toast['kind'], text: string, ms = 4000) {
		const id = next++;
		this.items = [...this.items, { id, kind, text }];
		setTimeout(() => this.dismiss(id), ms);
	}

	dismiss(id: number) {
		this.items = this.items.filter((t) => t.id !== id);
	}

	success = (text: string) => this.push('success', text);
	error = (text: string) => this.push('error', text, 6000);
	info = (text: string) => this.push('info', text);
}

export const toast = new ToastState();
