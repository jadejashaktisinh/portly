export const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
export const SLUG_PATTERN = /^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$/;
export const MIN_PASSWORD_LENGTH = 8;

export function validateSignupInput(value: unknown): string | null {
	if (typeof value !== "object" || value === null) {
		return "Invalid request body";
	}
	const { email, password, slug } = value as Record<string, unknown>;
	if (typeof email !== "string" || !EMAIL_PATTERN.test(email)) {
		return "A valid email is required";
	}
	if (typeof password !== "string" || password.length < MIN_PASSWORD_LENGTH) {
		return `Password must be at least ${MIN_PASSWORD_LENGTH} characters`;
	}
	if (typeof slug !== "string" || !SLUG_PATTERN.test(slug)) {
		return "Slug must be 1-40 lowercase letters, digits or hyphens";
	}
	return null;
}

export function validateLoginInput(value: unknown): string | null {
	if (typeof value !== "object" || value === null) {
		return "Invalid request body";
	}
	const { email, password } = value as Record<string, unknown>;
	if (typeof email !== "string" || email.length === 0) {
		return "Email is required";
	}
	if (typeof password !== "string" || password.length === 0) {
		return "Password is required";
	}
	return null;
}
