/**
 * Users API Client
 * Handles user-related API calls (username, profile basics)
 */

import { request } from '../base';

export interface UsernameAvailabilityResponse {
	available: boolean;
	reason?: string;
}

export interface SetUsernameResponse {
	success: boolean;
	username: string;
}

/**
 * Check if a username is available
 * @param username - The username to check
 * @returns Promise with availability status
 */
export async function checkUsernameAvailability(
	username: string
): Promise<UsernameAvailabilityResponse> {
	return request<UsernameAvailabilityResponse>(`/users/check-username/${encodeURIComponent(username)}`, {
		method: 'GET'
	});
}

/**
 * Set/update the user's username
 * @param username - The username to set
 * @returns Promise with success status
 */
export async function setUsername(username: string): Promise<SetUsernameResponse> {
	return request<SetUsernameResponse>('/users/me/username', {
		method: 'PATCH',
		body: { username }
	});
}
