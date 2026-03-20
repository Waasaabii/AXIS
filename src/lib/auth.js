import crypto from "node:crypto";

const HASH_PREFIX = "scrypt";

function base64UrlEncode(input) {
  return Buffer.from(input)
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

function base64UrlDecode(input) {
  const normalized = input.replace(/-/g, "+").replace(/_/g, "/");
  const padding = normalized.length % 4 === 0 ? "" : "=".repeat(4 - (normalized.length % 4));
  return Buffer.from(normalized + padding, "base64").toString("utf8");
}

function constantTimeEqualString(left, right) {
  const leftBuffer = Buffer.from(left);
  const rightBuffer = Buffer.from(right);

  if (leftBuffer.length !== rightBuffer.length) {
    return false;
  }

  return crypto.timingSafeEqual(leftBuffer, rightBuffer);
}

export function createPasswordHash(password) {
  const salt = crypto.randomBytes(16).toString("hex");
  const derived = crypto.scryptSync(password, salt, 64).toString("hex");
  return `${HASH_PREFIX}$${salt}$${derived}`;
}

function verifyPasswordHash(password, encodedHash) {
  const [prefix, salt, expectedHex] = encodedHash.split("$");
  if (prefix !== HASH_PREFIX || !salt || !expectedHex) {
    return false;
  }

  const actual = crypto.scryptSync(password, salt, 64);
  const expected = Buffer.from(expectedHex, "hex");

  if (actual.length !== expected.length) {
    return false;
  }

  return crypto.timingSafeEqual(actual, expected);
}

export function parseCookies(cookieHeader) {
  return cookieHeader
    .split(";")
    .map((part) => part.trim())
    .filter(Boolean)
    .reduce((cookies, item) => {
      const [name, ...rest] = item.split("=");
      cookies[name] = decodeURIComponent(rest.join("="));
      return cookies;
    }, {});
}

export function createAuthContext({ username, passwordHash, password, sessionSecret, sessionTtlHours }) {
  const ttlMs = Math.max(1, Number(sessionTtlHours || 12)) * 60 * 60 * 1000;
  const secret =
    sessionSecret ||
    crypto.createHash("sha256").update(`${username}:${passwordHash || password || "proxyrelay"}`).digest("hex");

  return {
    verifyCredentials(inputUsername, inputPassword) {
      if (inputUsername !== username) {
        return false;
      }

      if (passwordHash) {
        return verifyPasswordHash(inputPassword, passwordHash);
      }

      if (typeof password === "string") {
        return constantTimeEqualString(inputPassword, password);
      }

      return false;
    },

    createSessionCookie(sessionUsername) {
      const payload = {
        username: sessionUsername,
        exp: Date.now() + ttlMs,
        nonce: crypto.randomBytes(12).toString("hex")
      };
      const encodedPayload = base64UrlEncode(JSON.stringify(payload));
      const signature = crypto.createHmac("sha256", secret).update(encodedPayload).digest("hex");
      return `proxyrelay_session=${encodedPayload}.${signature}; Path=/; HttpOnly; SameSite=Strict; Max-Age=${Math.floor(
        ttlMs / 1000
      )}`;
    },

    clearSessionCookie() {
      return "proxyrelay_session=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0";
    },

    readSession(cookieValue) {
      if (!cookieValue) {
        return { valid: false };
      }

      const [encodedPayload, signature] = cookieValue.split(".");
      if (!encodedPayload || !signature) {
        return { valid: false };
      }

      const expectedSignature = crypto.createHmac("sha256", secret).update(encodedPayload).digest("hex");
      if (!constantTimeEqualString(signature, expectedSignature)) {
        return { valid: false };
      }

      try {
        const payload = JSON.parse(base64UrlDecode(encodedPayload));
        if (!payload.exp || payload.exp < Date.now()) {
          return { valid: false };
        }

        return { valid: true, username: payload.username };
      } catch {
        return { valid: false };
      }
    }
  };
}
