// Auth is now handled by the Go backend
// This file is kept for compatibility but doesn't do anything

export const auth = {
	handler: async () => new Response('Auth handled by backend', { status: 200 }),
	api: {}
};
