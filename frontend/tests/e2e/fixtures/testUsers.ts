/**
 * Test User Accounts
 *
 * Pre-configured test users for E2E testing.
 * These should match users in your test database.
 */

export interface TestUser {
	email: string;
	password: string;
	name: string;
	role?: 'admin' | 'user' | 'guest';
}

export const testUsers = {
	admin: {
		email: 'admin@test.businessos.com',
		password: 'TestAdmin123!',
		name: 'Admin Test User',
		role: 'admin' as const
	},

	regularUser: {
		email: 'user@test.businessos.com',
		password: 'TestUser123!',
		name: 'Regular Test User',
		role: 'user' as const
	},

	newUser: {
		email: `newuser-${Date.now()}@test.businessos.com`,
		password: 'NewUser123!',
		name: 'New Test User',
		role: 'user' as const
	},

	onboardingUser: {
		email: `onboarding-${Date.now()}@test.businessos.com`,
		password: 'Onboarding123!',
		name: 'Onboarding Test User',
		role: 'user' as const
	}
} as const;

/**
 * Get a test user by type
 */
export function getTestUser(type: keyof typeof testUsers): TestUser {
	return testUsers[type];
}

/**
 * Create a unique test user for isolation
 */
export function createUniqueTestUser(baseName = 'test'): TestUser {
	const timestamp = Date.now();
	return {
		email: `${baseName}-${timestamp}@test.businessos.com`,
		password: 'TestUser123!',
		name: `${baseName} User ${timestamp}`,
		role: 'user'
	};
}
