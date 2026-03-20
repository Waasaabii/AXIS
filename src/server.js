import http from "node:http";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseCookies } from "./lib/auth.js";
import { json, notFound, serveStatic, text } from "./lib/http.js";
import { createControlPlane } from "./services/control-plane.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const publicDir = path.resolve(__dirname, "..", "frontend", "dist");

function getRequestBody(request) {
  return new Promise((resolve, reject) => {
    let body = "";

    request.on("data", (chunk) => {
      body += chunk.toString("utf8");
      if (body.length > 1024 * 1024) {
        reject(new Error("请求体过大"));
      }
    });

    request.on("end", () => resolve(body));
    request.on("error", reject);
  });
}

function getStaticPath(urlPathname) {
  if (urlPathname === "/") {
    return "index.html";
  }

  return urlPathname.replace(/^\/+/, "");
}

export async function startServer() {
  const controlPlane = await createControlPlane();

  const server = http.createServer(async (request, response) => {
    const url = new URL(request.url, `http://${request.headers.host || "127.0.0.1"}`);
    const pathname = url.pathname;
    const method = request.method || "GET";
    const cookies = parseCookies(request.headers.cookie || "");
    const auth = controlPlane.getAuthContext();
    const session = auth.readSession(cookies.proxyrelay_session);

    try {
      if (pathname === "/api/session" && method === "GET") {
        if (!session.valid) {
          return json(response, 200, { authenticated: false });
        }

        return json(response, 200, {
          authenticated: true,
          user: { username: session.username }
        });
      }

      if (pathname === "/api/session" && method === "POST") {
        const rawBody = await getRequestBody(request);
        const payload = rawBody ? JSON.parse(rawBody) : {};
        const result = await controlPlane.login(payload.username, payload.password);

        if (!result.ok) {
          return json(response, 401, { ok: false, error: result.error });
        }

        const cookie = auth.createSessionCookie(result.user.username);
        response.setHeader("Set-Cookie", cookie);
        return json(response, 200, {
          ok: true,
          user: { username: result.user.username }
        });
      }

      if (pathname === "/api/session" && method === "DELETE") {
        response.setHeader("Set-Cookie", auth.clearSessionCookie());
        return json(response, 200, { ok: true });
      }

      if (pathname.startsWith("/api/")) {
        if (!session.valid) {
          return json(response, 401, { ok: false, error: "未登录或会话已过期" });
        }

        if (pathname === "/api/status" && method === "GET") {
          return json(response, 200, await controlPlane.getStatus());
        }

        if (pathname === "/api/config" && method === "GET") {
          return json(response, 200, await controlPlane.getConfig());
        }

        if (pathname === "/api/config" && method === "PUT") {
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.saveConfig(payload.config || payload));
        }

        if (pathname === "/api/providers" && method === "GET") {
          return json(response, 200, await controlPlane.getProviders());
        }

        if (pathname === "/api/groups" && method === "GET") {
          return json(response, 200, await controlPlane.getGroups());
        }

        if (pathname === "/api/listeners" && method === "GET") {
          return json(response, 200, await controlPlane.getListeners());
        }

        if (pathname === "/api/events" && method === "GET") {
          return json(response, 200, await controlPlane.getEvents());
        }

        if (pathname === "/api/rendered-config" && method === "GET") {
          return json(response, 200, await controlPlane.getRenderedConfig());
        }

        if (pathname === "/api/subscription-test" && method === "POST") {
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.testSubscription(payload));
        }

        if (pathname === "/api/controller" && method === "GET") {
          return json(response, 200, await controlPlane.getControllerStatus());
        }

        if (pathname === "/api/runtime-preflight" && method === "GET") {
          return json(response, 200, await controlPlane.getRuntimePreflight());
        }

        if (pathname === "/api/controller/probe" && method === "POST") {
          return json(response, 200, await controlPlane.probeController());
        }

        if (pathname === "/api/reload" && method === "POST") {
          return json(response, 200, await controlPlane.reloadConfig());
        }

        if (pathname === "/api/subscriptions" && method === "POST") {
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.addSubscription(payload));
        }

        if (pathname.startsWith("/api/subscriptions/") && pathname.endsWith("/toggle") && method === "POST") {
          const name = decodeURIComponent(pathname.split("/")[3]);
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.toggleSubscription(name, payload.enabled));
        }

        if (pathname.startsWith("/api/subscriptions/") && method === "DELETE") {
          const name = decodeURIComponent(pathname.split("/")[3]);
          return json(response, 200, await controlPlane.removeSubscription(name));
        }

        if (pathname === "/api/egress-groups" && method === "POST") {
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.addEgressGroup(payload));
        }

        if (pathname.startsWith("/api/egress-groups/") && pathname.endsWith("/update") && method === "PUT") {
          const name = decodeURIComponent(pathname.split("/")[3]);
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.updateEgressGroup(name, payload));
        }

        if (pathname.startsWith("/api/egress-groups/") && method === "DELETE") {
          const name = decodeURIComponent(pathname.split("/")[3]);
          return json(response, 200, await controlPlane.removeEgressGroup(name));
        }

        if (pathname === "/api/listeners" && method === "POST") {
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.addListener(payload));
        }

        if (pathname.startsWith("/api/listeners/") && pathname.endsWith("/update") && method === "PUT") {
          const name = decodeURIComponent(pathname.split("/")[3]);
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.updateListener(name, payload));
        }

        if (pathname.startsWith("/api/listeners/") && method === "DELETE") {
          const name = decodeURIComponent(pathname.split("/")[3]);
          return json(response, 200, await controlPlane.removeListener(name));
        }

        if (pathname.startsWith("/api/providers/") && pathname.endsWith("/refresh") && method === "POST") {
          const providerName = decodeURIComponent(pathname.split("/")[3]);
          return json(response, 200, await controlPlane.refreshProvider(providerName));
        }

        if (pathname.startsWith("/api/groups/") && pathname.endsWith("/select") && method === "POST") {
          const groupName = decodeURIComponent(pathname.split("/")[3]);
          const rawBody = await getRequestBody(request);
          const payload = rawBody ? JSON.parse(rawBody) : {};
          return json(response, 200, await controlPlane.selectGroup(groupName, payload.proxyName));
        }

        if (pathname.startsWith("/api/groups/") && pathname.endsWith("/healthcheck") && method === "POST") {
          const groupName = decodeURIComponent(pathname.split("/")[3]);
          return json(response, 200, await controlPlane.runHealthcheck(groupName));
        }

        return notFound(response);
      }

      const candidate = getStaticPath(pathname);
      const served = await serveStatic(response, publicDir, candidate);
      if (served) {
        return;
      }

      const fallbackHtml = await readFile(path.join(publicDir, "index.html"), "utf8");
      return text(response, 200, fallbackHtml, "text/html; charset=utf-8");
    } catch (error) {
      console.error("[http]", error);
      return json(response, 500, {
        ok: false,
        error: error instanceof Error ? error.message : "服务内部错误"
      });
    }
  });

  const { host, port } = controlPlane.getServerConfig();
  server.listen(port, host, () => {
    console.log(`[proxyrelay] 控制面已启动: http://${host}:${port}`);
  });
}
