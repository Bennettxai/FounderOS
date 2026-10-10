// DEPRECATED: This store is no longer used.
// Better Auth handles authentication via $lib/auth-client
//
// Use the following instead:
//   import { useSession, signIn, signOut } from '$lib/auth-client';
//
// In Svelte components:
//   const session = useSession();
//   $session.data?.user - current user
//   $session.isPending - loading state

export {};
