/** Globe projection math (FounderOS v1 components/adpilot/Globe.tsx). */
import type { GeoPoint } from './types';

export type Vec = { x: number; y: number; z: number };

/** Unit vector on the sphere: +y north, +z toward the viewer at yaw 0. */
export function latLngToVec(lat: number, lng: number): Vec {
	const phi = (lat * Math.PI) / 180;
	const lambda = (lng * Math.PI) / 180;
	return { x: Math.cos(phi) * Math.cos(lambda), y: Math.sin(phi), z: -Math.cos(phi) * Math.sin(lambda) };
}

/** Rotate about Y (yaw), then about X (pitch). */
export function rotate(v: Vec, yaw: number, pitch: number): Vec {
	const cy = Math.cos(yaw);
	const sy = Math.sin(yaw);
	const x1 = v.x * cy + v.z * sy;
	const z1 = -v.x * sy + v.z * cy;
	const cx = Math.cos(pitch);
	const sx = Math.sin(pitch);
	return { x: x1, y: v.y * cx - z1 * sx, z: v.y * sx + z1 * cx };
}

/** The labeled cities: the top six by leads. */
export function globeChips(geo: GeoPoint[], n = 6): GeoPoint[] {
	return [...geo].sort((a, b) => b.leads - a.leads).slice(0, n);
}
