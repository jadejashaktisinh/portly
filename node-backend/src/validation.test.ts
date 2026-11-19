import test from "node:test";
import assert from "node:assert/strict";
import { validateLoginInput, validateSignupInput } from "./validation.js";

test("signup accepts valid input", () => {
	assert.equal(
		validateSignupInput({
			email: "user@example.com",
			password: "long-enough",
			slug: "my-slug",
		}),
		null,
	);
});

test("signup rejects invalid email, short password and bad slug", () => {
	assert.match(
		validateSignupInput({ email: "bad", password: "long-enough", slug: "ok" }) ?? "",
		/email/i,
	);
	assert.match(
		validateSignupInput({ email: "a@b.co", password: "short", slug: "ok" }) ?? "",
		/password/i,
	);
	assert.match(
		validateSignupInput({ email: "a@b.co", password: "long-enough", slug: "Bad Slug" }) ?? "",
		/slug/i,
	);
});

test("login requires email and password", () => {
	assert.equal(validateLoginInput({ email: "a@b.co", password: "secret" }), null);
	assert.match(validateLoginInput({ password: "secret" }) ?? "", /email/i);
	assert.match(validateLoginInput({ email: "a@b.co" }) ?? "", /password/i);
});
