import { createPasswordHash } from "./lib/auth.js";
import { createControlPlane } from "./services/control-plane.js";
import { formatRuntimePreflightForCli } from "./services/runtime-preflight.js";
import { startServer } from "./server.js";

function printUsage() {
  console.log("用法:");
  console.log("  node src/index.js serve");
  console.log("  node src/index.js hash-password <password>");
  console.log("  node src/index.js preflight [--json]");
}

async function main() {
  const [command, arg] = process.argv.slice(2);

  if (!command || command === "serve") {
    await startServer();
    return;
  }

  if (command === "hash-password") {
    if (!arg) {
      console.error("请提供需要哈希的密码");
      process.exitCode = 1;
      return;
    }

    console.log(createPasswordHash(arg));
    return;
  }

  if (command === "preflight") {
    const outputJson = arg === "--json";
    const controlPlane = await createControlPlane({ skipStartupRefresh: true });
    const snapshot = await controlPlane.getRuntimePreflight();

    if (outputJson) {
      console.log(JSON.stringify(snapshot, null, 2));
    } else {
      process.stdout.write(formatRuntimePreflightForCli(snapshot));
    }

    process.exitCode = snapshot.ready ? 0 : 1;
    return;
  }

  printUsage();
  process.exitCode = 1;
}

main().catch((error) => {
  console.error("[fatal]", error);
  process.exitCode = 1;
});
