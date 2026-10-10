import { fireEvent, render } from '@testing-library/svelte';
import { beforeEach, describe, expect, it } from 'vitest';
import { get } from 'svelte/store';
import Window from './Window.svelte';
import { windowStore, type WindowState } from '$lib/stores/windowStore';

describe('Window maximize bounds', () => {
	beforeEach(() => {
		localStorage.clear();
		windowStore.reset();
	});

	it('uses explicit maximize bounds instead of the backing canvas size', () => {
		const windowState: WindowState = {
			id: 'win-canvas-test',
			module: 'knowledge',
			title: 'Knowledge',
			x: 100,
			y: 120,
			width: 640,
			height: 420,
			minWidth: 320,
			minHeight: 220,
			minimized: false,
			maximized: true
		};

		const { container } = render(Window, {
			props: {
				window: windowState,
				focused: true,
				zIndex: 10,
				workspaceWidth: 100000,
				workspaceHeight: 100000,
				maximizeBounds: { x: 25, y: 30, width: 1200, height: 760 }
			}
		});

		const dialog = container.querySelector('[role="dialog"]') as HTMLElement;
		const style = dialog.getAttribute('style') ?? '';

		expect(style).toContain('left: 25px');
		expect(style).toContain('top: 30px');
		expect(style).toContain('width: 1200px');
		expect(style).toContain('height: 760px');
		expect(style).not.toContain('100000px');
	});

	it('ignores stale maximized state on unbounded canvas desktops', () => {
		const windowState: WindowState = {
			id: 'win-canvas-terminal',
			module: 'terminal',
			title: 'Terminal',
			x: -1200,
			y: 800,
			width: 700,
			height: 500,
			minWidth: 320,
			minHeight: 220,
			minimized: false,
			maximized: true,
			previousBounds: {
				x: -1200,
				y: 800,
				width: 700,
				height: 500
			}
		};

		const { container } = render(Window, {
			props: {
				window: windowState,
				focused: true,
				zIndex: 10,
				workspaceWidth: 100000,
				workspaceHeight: 100000,
				unbounded: true,
				maximizeBounds: { x: -20, y: -20, width: 8000, height: 5000 }
			}
		});

		const dialog = container.querySelector('[role="dialog"]') as HTMLElement;
		const maximize = container.querySelector('.control-button.maximize') as HTMLButtonElement;
		const style = dialog.getAttribute('style') ?? '';

		expect(style).toContain('left: -1200px');
		expect(style).toContain('top: 800px');
		expect(style).toContain('width: 700px');
		expect(style).toContain('height: 500px');
		expect(style).not.toContain('8000px');
		expect(maximize.disabled).toBe(true);
	});

	it('does not toggle maximize from the green button on unbounded canvas desktops', async () => {
		windowStore.openWindow('terminal', { x: -400, y: 240 });
		const stateWindow = get(windowStore).windows.find((item) => item.module === 'terminal');

		const { container } = render(Window, {
			props: {
				window: stateWindow!,
				focused: true,
				zIndex: 10,
				workspaceWidth: 100000,
				workspaceHeight: 100000,
				unbounded: true
			}
		});

		const maximize = container.querySelector('.control-button.maximize') as HTMLButtonElement;
		await fireEvent.click(maximize);

		expect(get(windowStore).windows.find((item) => item.id === stateWindow!.id)?.maximized).toBe(false);
	});
});
