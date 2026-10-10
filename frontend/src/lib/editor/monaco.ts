/**
 * Monaco editor setup with worker configuration and custom theme.
 *
 * WARNING: This module references `self.MonacoEnvironment` at module scope and
 * must ONLY be loaded via dynamic import (e.g. `await import('./monaco')` inside
 * `onMount`). A static import will crash during SSR because `self` is undefined
 * in Node.js.
 */
import * as monaco from 'monaco-editor';

// @ts-ignore — Vite worker imports
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
// @ts-ignore
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker';
// @ts-ignore
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';
// @ts-ignore
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
// @ts-ignore
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';

import { businessOSDark } from './themes/businessos-dark';

self.MonacoEnvironment = {
	getWorker(_: string, label: string) {
		switch (label) {
			case 'json':
				return new jsonWorker();
			case 'css':
			case 'scss':
			case 'less':
				return new cssWorker();
			case 'html':
			case 'handlebars':
			case 'razor':
				return new htmlWorker();
			case 'typescript':
			case 'javascript':
				return new tsWorker();
			default:
				return new editorWorker();
		}
	},
};

// Register custom theme
monaco.editor.defineTheme('businessos-dark', businessOSDark);

export default monaco;
