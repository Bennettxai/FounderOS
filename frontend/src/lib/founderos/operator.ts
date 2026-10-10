/** Who the views address: the signed-in user's first name, else their email's
 *  local part, else a neutral "there". The (founderos) layout load puts the
 *  session user on page.data. */
export function operatorName(user: { name?: unknown; email?: unknown } | undefined): string {
	const name = typeof user?.name === 'string' ? user.name.trim() : '';
	if (name) return name.split(/\s+/)[0];
	const email = typeof user?.email === 'string' ? user.email : '';
	return email.includes('@') ? email.slice(0, email.indexOf('@')) : 'there';
}
