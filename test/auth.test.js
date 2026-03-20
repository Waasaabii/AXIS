import test from "node:test";
import assert from "node:assert/strict";
import { createAuthContext, createPasswordHash } from "../src/lib/auth.js";

test("password hash can be verified", () => {
  const hash = createPasswordHash("secret-pass");
  const auth = createAuthContext({
    username: "admin",
    passwordHash: hash,
    sessionSecret: "test-secret",
    sessionTtlHours: 1
  });

  assert.equal(auth.verifyCredentials("admin", "secret-pass"), true);
  assert.equal(auth.verifyCredentials("admin", "wrong"), false);
  assert.equal(auth.verifyCredentials("demo", "secret-pass"), false);
});

test("session cookie can be issued and verified", () => {
  const auth = createAuthContext({
    username: "admin",
    password: "change-me",
    sessionSecret: "test-secret",
    sessionTtlHours: 1
  });

  const cookie = auth.createSessionCookie("admin");
  const value = cookie.split(";")[0].split("=")[1];
  const session = auth.readSession(value);

  assert.equal(session.valid, true);
  assert.equal(session.username, "admin");
});
