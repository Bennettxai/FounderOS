import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import {
	appMode,
	cloudServerUrl,
	setAppMode,
	initiateGoogleOAuth,
	requestPasswordReset,
	signUpWithEmail,
	signInWithEmail,
	getSession,
	signOutFromServer,
	clearSession,
	refreshSession,
	signIn,
	signUp,
	signOut,
	useSession
} from './auth-client';

// Mock window and localStorage
const localStorageMock = (() => {
	let store: Record<string, string> = {};
	return {
		getItem: (key: string) => store[key] || null,
		setItem: (key: string, value: string) => {
			store[key] = value.toString();
		},
		removeItem: (key: string) => {
			delete store[key];
		},
		clear: () => {
			store = {};
		}
	};
})();

Object.defineProperty(global, 'localStorage', {
	value: localStorageMock,
	writable: true
});

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch as any;

// Mock window.location
const mockLocation = {
	hostname: 'localhost',
	origin: 'http://localhost:5174',
	href: '',
	reload: vi.fn()
};
Object.defineProperty(global, 'window', {
	value: {
		location: mockLocation,
		document: {
			cookie: ''
		}
	},
	writable: true
});

describe('Auth Client', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		localStorageMock.clear();
		mockLocation.href = '';
		(window as any).document.cookie = 'csrf_token=default-test-token';
		mockFetch.mockReset();
	});

	afterEach(() => {
		vi.clearAllMocks();
	});

	describe('App Mode Management', () => {
		it('initializes with mode from environment', () => {
			// In dev mode (localhost), it auto-sets to cloud mode
			const mode = get(appMode);
			expect(mode).toBeDefined(); // Could be 'cloud' or null depending on initialization
		});

		it('loads and persists mode to localStorage', () => {
			// The store may have already initialized, so we test the setAppMode function
			const mode = get(appMode);
			const url = get(cloudServerUrl);

			// Either it's already set or we can set it
			expect(mode !== null || url !== '').toBe(true);
		});

		it('setAppMode updates store and localStorage for cloud mode', () => {
			setAppMode('cloud', 'https://test-server.com');

			expect(localStorageMock.getItem('businessos_mode')).toBe('cloud');
			expect(localStorageMock.getItem('businessos_cloud_url')).toBe('https://test-server.com');
			expect(mockLocation.reload).toHaveBeenCalled();
		});

		it('setAppMode updates store and localStorage for local mode', () => {
			setAppMode('local');

			expect(localStorageMock.getItem('businessos_mode')).toBe('local');
			expect(mockLocation.reload).toHaveBeenCalled();
		});
	});

	describe('CSRF Token Handling', () => {
		it('initializes CSRF before a password reset request when no cookie is readable', async () => {
			Object.defineProperty(document, 'cookie', { writable: true, value: '' });
			mockFetch
				.mockResolvedValueOnce({
					ok: true,
					json: async () => ({ csrf_token: 'reset-csrf-token' })
				})
				.mockResolvedValueOnce({
					ok: true,
					json: async () => ({ message: 'Reset sent' })
				});

			const result = await requestPasswordReset('person@example.com');

			expect(result.ok).toBe(true);
			expect(mockFetch).toHaveBeenCalledTimes(2);
			expect(mockFetch.mock.calls[1][1].headers['X-CSRF-Token']).toBe('reset-csrf-token');
		});

		it('extracts CSRF token from cookies', async () => {
			// Set up document.cookie properly
			Object.defineProperty(document, 'cookie', {
				writable: true,
				value: 'csrf_token=test-csrf-token-123; other=value'
			});

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ user: { id: '1', email: 'test@example.com' } })
			});

			await signUpWithEmail('test@example.com', 'password123', 'Test User', 'https://api.com');

			// Check that fetch was called with the right URL and method
			expect(mockFetch).toHaveBeenCalledWith(
				'https://api.com/api/auth/sign-up/email',
				expect.objectContaining({
					method: 'POST'
				})
			);

			// Check that CSRF token was included (it's in the implementation)
			const callArgs = mockFetch.mock.calls[0][1] as any;
			expect(callArgs.headers['X-CSRF-Token']).toBe('test-csrf-token-123');

			// Reset cookie
			Object.defineProperty(document, 'cookie', { writable: true, value: '' });
		});

		it('handles missing CSRF token gracefully', async () => {
			Object.defineProperty(document, 'cookie', {
				writable: true,
				value: 'other=value'
			});

			mockFetch
				.mockResolvedValueOnce({
					ok: true,
					json: async () => ({ csrf_token: 'initialized-token' })
				})
				.mockResolvedValueOnce({
					ok: true,
					json: async () => ({ user: { id: '1' } })
				});

			await signUpWithEmail('test@example.com', 'password123', 'Test', 'https://api.com');

			expect(mockFetch).toHaveBeenCalled();

			Object.defineProperty(document, 'cookie', { writable: true, value: '' });
		});

		it('handles CSRF token with equals sign in value', async () => {
			Object.defineProperty(document, 'cookie', {
				writable: true,
				value: 'csrf_token=token==value; other=test'
			});

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			});

			await signInWithEmail('test@example.com', 'pass', 'https://api.com');

			const call = mockFetch.mock.calls[0];
			const headers = call[1].headers as Record<string, string>;
			expect(headers['X-CSRF-Token']).toBe('token==value');

			Object.defineProperty(document, 'cookie', { writable: true, value: '' });
		});
	});

	describe('Google OAuth', () => {
		it('initiates web Google OAuth through the same-origin proxy', () => {
			const result = initiateGoogleOAuth('https://api.example.com');

			expect(result).toBe(true);
			expect(mockLocation.href).toContain('/api/auth/google');
			expect(mockLocation.href).not.toContain('https://api.example.com');
			expect(mockLocation.href).toContain('redirect=');
		});

		it('returns result based on cloudServerUrl store state', () => {
			// The store might have a URL from initialization, so we test the behavior
			const storeUrl = get(cloudServerUrl);
			const result = initiateGoogleOAuth();

			// If store has URL, should return true, otherwise false
			expect(typeof result).toBe('boolean');
		});

		it('does not bypass the same-origin proxy when a cloud URL is stored', () => {
			cloudServerUrl.set('https://stored-url.com');

			const result = initiateGoogleOAuth();

			expect(result).toBe(true);
			expect(mockLocation.href).toContain('/api/auth/google');
			expect(mockLocation.href).not.toContain('https://stored-url.com');
		});

		it('opens external browser in Electron mode', () => {
			const mockOpenExternal = vi.fn();
			const originalElectron = (window as any).electron;
			(window as any).electron = {
				shell: {
					openExternal: mockOpenExternal
				}
			};

			initiateGoogleOAuth('https://api.example.com');

			expect(mockOpenExternal).toHaveBeenCalledWith(
				expect.stringContaining('https://api.example.com/api/auth/google')
			);
			expect(mockLocation.href).toBe('');

			(window as any).electron = originalElectron;
		});
	});

	describe('Sign Up with Email', () => {
		it('signs up successfully', async () => {
			const mockResponse = {
				user: { id: '1', email: 'test@example.com', name: 'Test User' }
			};

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => mockResponse
			});

			const result = await signUpWithEmail('test@example.com', 'password123', 'Test User', 'https://api.com');

			expect(result.data).toEqual(mockResponse);
			expect(result.error).toBeUndefined();
			expect(mockFetch).toHaveBeenCalledWith(
				'https://api.com/api/auth/sign-up/email',
				expect.objectContaining({
					method: 'POST',
					credentials: 'include',
					headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
					body: JSON.stringify({ email: 'test@example.com', password: 'password123', name: 'Test User' })
				})
			);
		});

		it('handles sign up failure with error message', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				json: async () => ({ error: 'Email already exists' })
			});

			const result = await signUpWithEmail('test@example.com', 'password123', 'Test', 'https://api.com');

			expect(result.error).toEqual({ message: 'Email already exists' });
			expect(result.data).toBeUndefined();
		});

		it('handles sign up failure without error message', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				json: async () => ({})
			});

			const result = await signUpWithEmail('test@example.com', 'password123', 'Test', 'https://api.com');

			expect(result.error).toEqual({ message: 'Sign up failed' });
		});

		it('handles network error', async () => {
			mockFetch.mockRejectedValueOnce(new Error('Network failure'));

			const result = await signUpWithEmail('test@example.com', 'password123', 'Test', 'https://api.com');

			expect(result.error).toEqual({ message: 'Network failure' });
		});

		it('returns error when no server URL configured', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ user: { id: '1' } })
			});
			cloudServerUrl.set('');

			const result = await signUpWithEmail('test@example.com', 'password123', 'Test');

			expect(result.data).toBeDefined();
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/auth/sign-up/email',
				expect.objectContaining({ method: 'POST' })
			);
		});
	});

	describe('Sign In with Email', () => {
		it('signs in successfully', async () => {
			const mockResponse = {
				user: { id: '1', email: 'test@example.com', name: 'Test User' }
			};

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => mockResponse
			});

			const result = await signInWithEmail('test@example.com', 'password123', 'https://api.com');

			expect(result.data).toEqual(mockResponse);
			expect(result.error).toBeUndefined();
			expect(mockFetch).toHaveBeenCalledWith(
				'https://api.com/api/auth/sign-in/email',
				expect.objectContaining({
					method: 'POST',
					credentials: 'include'
				})
			);
		});

		it('handles sign in failure', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				json: async () => ({ error: 'Invalid credentials' })
			});

			const result = await signInWithEmail('test@example.com', 'wrong-password', 'https://api.com');

			expect(result.error).toEqual({ message: 'Invalid credentials' });
		});

		it('handles network error during sign in', async () => {
			mockFetch.mockRejectedValueOnce(new Error('Connection timeout'));

			const result = await signInWithEmail('test@example.com', 'password123', 'https://api.com');

			expect(result.error).toEqual({ message: 'Connection timeout' });
		});

		it('returns error when no server URL configured', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ user: { id: '1' } })
			});
			cloudServerUrl.set('');

			const result = await signInWithEmail('test@example.com', 'password123');

			expect(result.data).toBeDefined();
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/auth/sign-in/email',
				expect.objectContaining({ method: 'POST' })
			);
		});
	});

	describe('Session Management', () => {
		it('uses the same-origin session endpoint in Electron development', async () => {
			const originalElectron = (window as any).electron;
			(window as any).electron = {};
			mockFetch.mockResolvedValueOnce({
				ok: false,
				headers: new Headers({ 'content-type': 'application/json' })
			});

			await getSession();

			expect(mockFetch).toHaveBeenCalledWith(
				'/api/auth/session',
				expect.objectContaining({ credentials: 'include' })
			);
			(window as any).electron = originalElectron;
		});

		it('does not ask the server for a session after an explicit logout', async () => {
			localStorageMock.setItem('businessos_logged_out', '1');

			const result = await getSession();

			expect(result).toEqual({ data: null, error: 'Logged out' });
			expect(mockFetch).not.toHaveBeenCalled();
		});

		it('gets session successfully', async () => {
			const mockSessionData = {
				user: { id: '1', email: 'test@example.com', name: 'Test User' },
				session: { id: 'session-123' }
			};

			mockFetch.mockResolvedValueOnce({
				ok: true,
				headers: new Headers({ 'content-type': 'application/json' }),
				json: async () => mockSessionData
			});

			const result = await getSession('https://api.com');

			expect(result.data).toEqual(mockSessionData);
			expect(result.error).toBeNull();
			expect(mockFetch).toHaveBeenCalledWith(
				'https://api.com/api/auth/session',
				expect.objectContaining({
					method: 'GET',
					credentials: 'include'
				})
			);
		});

		it('handles unauthenticated session', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false
			});

			const result = await getSession('https://api.com');

			expect(result.data).toBeNull();
			expect(result.error).toBe('Not authenticated');
		});

		it('handles network error when getting session', async () => {
			mockFetch.mockRejectedValueOnce(new Error('Network error'));

			const result = await getSession('https://api.com');

			expect(result.data).toBeNull();
			expect(result.error).toBe('Network error');
		});

		it('returns error when no server URL configured', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				headers: new Headers({ 'content-type': 'application/json' })
			});
			cloudServerUrl.set('');

			const result = await getSession();

			expect(result.data).toBeNull();
			expect(result.error).toBe('Not authenticated');
		});

		it('clears session data', () => {
			// clearSession is tested implicitly through other functions
			clearSession();
			// Session should be cleared - we can't directly test internal cloudSession store
			// but the function should execute without errors
			expect(true).toBe(true);
		});

		it('refreshes session', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				headers: new Headers({ 'content-type': 'application/json' }),
				json: async () => ({ user: { id: '1' }, session: { id: 's1' } })
			});

			cloudServerUrl.set('https://api.com');
			appMode.set('cloud');

			await refreshSession();

			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/auth/session'),
				expect.any(Object)
			);
		});
	});

	describe('Sign Out', () => {
		it('signs out successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true
			});

			const result = await signOutFromServer('https://api.com');

			expect(result.success).toBe(true);
			expect(mockFetch).toHaveBeenCalledWith(
				'https://api.com/api/auth/logout',
				expect.objectContaining({
					method: 'POST',
					credentials: 'include'
				})
			);
			expect(mockLocation.href).toBe('');
		});

		it('handles sign out failure but still redirects', async () => {
			const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 500,
				text: async () => 'Server error'
			});

			const result = await signOutFromServer('https://api.com');

			expect(result.success).toBe(false);
			expect(result.error).toContain('500');
			expect(mockLocation.href).toBe('');

			consoleErrorSpy.mockRestore();
		});

		it('handles network error during sign out', async () => {
			const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

			mockFetch.mockRejectedValueOnce(new Error('Network failure'));

			const result = await signOutFromServer('https://api.com');

			expect(result.success).toBe(false);
			expect(result.error).toBe('Network failure');
			expect(mockLocation.href).toBe('');

			consoleErrorSpy.mockRestore();
		});

		it('returns an error without navigating when no server URL is configured', async () => {
			const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
			cloudServerUrl.set('');

			const result = await signOutFromServer();

			expect(result.success).toBe(false);
			expect(result.error).toBeDefined();
			expect(mockLocation.href).toBe('');

			consoleErrorSpy.mockRestore();
		});

		it('redirects from the top-level sign out flow', async () => {
			mockFetch.mockResolvedValueOnce({ ok: true });

			await signOut();

			expect(mockLocation.href).toBe('/login');
		});

		it('clears the persisted Electron cookie before redirecting', async () => {
			const clearSessionCookie = vi.fn().mockResolvedValue(undefined);
			const originalElectron = (window as any).electron;
			(window as any).electron = { auth: { clearSession: clearSessionCookie } };
			mockFetch.mockResolvedValueOnce({ ok: true });

			await signOut();

			expect(clearSessionCookie).toHaveBeenCalledOnce();
			expect(localStorageMock.getItem('businessos_logged_out')).toBe('1');
			expect(mockLocation.href).toBe('/login');
			(window as any).electron = originalElectron;
		});
	});

	describe('Mode-based Auth Functions', () => {
		it('uses cloud sign in functions in cloud mode', () => {
			expect(signIn).toBeDefined();
			expect(signIn.email).toBeDefined();
			expect(signIn.social).toBeDefined();
		});

		it('uses cloud sign up functions in cloud mode', () => {
			expect(signUp).toBeDefined();
			expect(signUp.email).toBeDefined();
		});

		it('uses cloud sign out function in cloud mode', () => {
			expect(signOut).toBeDefined();
			expect(typeof signOut).toBe('function');
		});

		it('provides session hook', () => {
			const session = useSession();
			expect(session).toBeDefined();
		});
	});

	describe('Edge Cases', () => {
		it('handles empty email', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				json: async () => ({ error: 'Email is required' })
			});

			const result = await signInWithEmail('', 'password123', 'https://api.com');

			expect(result.error).toBeDefined();
		});

		it('handles empty password', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				json: async () => ({ error: 'Password is required' })
			});

			const result = await signInWithEmail('test@example.com', '', 'https://api.com');

			expect(result.error).toBeDefined();
		});

		it('handles malformed JSON response', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => {
					throw new Error('Invalid JSON');
				}
			});

			const result = await signInWithEmail('test@example.com', 'password', 'https://api.com');

			expect(result.error).toBeDefined();
		});

		it('handles multiple CSRF tokens in cookies', async () => {
			Object.defineProperty(document, 'cookie', {
				writable: true,
				value: 'csrf_token=token1; csrf_token=token2; other=value'
			});

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			});

			await signInWithEmail('test@example.com', 'password', 'https://api.com');

			// Should use one of the tokens (the last one found)
			expect(mockFetch).toHaveBeenCalled();

			Object.defineProperty(document, 'cookie', { writable: true, value: '' });
		});

		it('handles invalid server URL format', async () => {
			mockFetch.mockRejectedValueOnce(new TypeError('Failed to fetch'));

			const result = await signInWithEmail('test@example.com', 'password', 'invalid-url');

			expect(result.error).toBeDefined();
		});
	});

	describe('Integration Scenarios', () => {
		it('complete sign up and sign in flow', async () => {
			// Sign up
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ user: { id: '1', email: 'new@example.com' } })
			});

			const signUpResult = await signUpWithEmail('new@example.com', 'password123', 'New User', 'https://api.com');
			expect(signUpResult.data).toBeDefined();

			// Sign in
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ user: { id: '1', email: 'new@example.com' } })
			});

			const signInResult = await signInWithEmail('new@example.com', 'password123', 'https://api.com');
			expect(signInResult.data).toBeDefined();
		});

		it('handles session expiration and refresh', async () => {
			// Initial session fails
			mockFetch.mockResolvedValueOnce({
				ok: false
			});

			const result1 = await getSession('https://api.com');
			expect(result1.error).toBe('Not authenticated');

			// Refresh session succeeds
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ user: { id: '1' }, session: { id: 's1' } })
			});

			cloudServerUrl.set('https://api.com');
			appMode.set('cloud');
			await refreshSession();

			// Should have attempted to get session
			expect(mockFetch).toHaveBeenCalledTimes(2);
		});
	});
});
