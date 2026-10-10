/** Compose the Conductor emblem's classes (components/ConductorEmblem.tsx):
 *  `thinking` is the one bit its animation keys off. */
export function conductorEmblemClasses(thinking: boolean, className = ''): string {
	return ['conductor-emblem', thinking ? 'thinking' : '', className].filter(Boolean).join(' ');
}
