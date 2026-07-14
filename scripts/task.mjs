import { spawn, spawnSync } from "node:child_process";
import { copyFileSync, existsSync, rmSync } from "node:fs";
import { join } from "node:path";

const root = new URL("..", import.meta.url).pathname.replace(/^\/(.:)/, "$1");
const task = process.argv[2];
const isWindows = process.platform === "win32";

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd || root,
    env: { ...process.env, ...options.env },
    shell: isWindows,
    stdio: "inherit"
  });
  if (result.status !== 0) process.exit(result.status ?? 1);
}

function runPnpm(args, options = {}) {
  run("corepack", ["pnpm", ...args], options);
}

function runConcurrent(processes) {
  const children = processes.map(({ command, args, cwd = root }) => spawn(command, args, {
    cwd,
    env: process.env,
    shell: isWindows,
    stdio: "inherit"
  }));
  let stopping = false;
  const stop = exitCode => {
    if (stopping) return;
    stopping = true;
    children.forEach(child => {
      if (!child.killed) child.kill();
    });
    process.exit(exitCode);
  };
  process.on("SIGINT", () => stop(130));
  process.on("SIGTERM", () => stop(143));
  children.forEach(child => {
    child.on("error", error => {
      console.error(error);
      stop(1);
    });
    child.on("exit", code => {
      if (!stopping) stop(code ?? 1);
    });
  });
}

const tasks = {
  setup() {
    if (!existsSync(join(root, ".env"))) copyFileSync(join(root, ".env.example"), join(root, ".env"));
    runPnpm(["install", "--frozen-lockfile=false"]);
    run("go", ["mod", "download"], { cwd: join(root, "services/core-api") });
  },
  migrate() {
    run("go", ["run", "./cmd/migrate", "up"], { cwd: join(root, "services/core-api") });
  },
  seed() {
    run("go", ["run", "./cmd/migrate", "seed"], { cwd: join(root, "services/core-api") });
  },
  dev() {
    run("docker", ["compose", "up", "-d", "postgres"]);
    run("go", ["run", "./cmd/migrate", "up"], { cwd: join(root, "services/core-api") });
    run("go", ["run", "./cmd/migrate", "seed"], { cwd: join(root, "services/core-api") });
    runConcurrent([
      { command: "go", args: ["run", "./cmd/api"], cwd: join(root, "services/core-api") },
      { command: "corepack", args: ["pnpm", "--filter", "@opportunity-os/admin-web", "dev"] },
      { command: "corepack", args: ["pnpm", "--filter", "@opportunity-os/operator-console", "dev"] }
    ]);
  },
  test() {
    run("docker", ["compose", "up", "-d", "postgres"]);
    run("go", ["run", "./cmd/migrate", "up"], { cwd: join(root, "services/core-api") });
    run("go", ["test", "./..."], { cwd: join(root, "services/core-api") });
    runPnpm(["-r", "--if-present", "test"]);
  },
  lint() {
    run("go", ["vet", "./..."], { cwd: join(root, "services/core-api") });
    runPnpm(["-r", "--if-present", "typecheck"]);
  },
  build() {
    run("go", ["build", "./..."], { cwd: join(root, "services/core-api") });
    runPnpm(["-r", "--if-present", "build"]);
  },
  e2e() {
    run("docker", ["compose", "up", "-d", "postgres"]);
    run("go", ["run", "./cmd/migrate", "up"], { cwd: join(root, "services/core-api") });
    run("go", ["test", "./tests", "-run", "TestNeutralEndToEnd", "-v"], { cwd: join(root, "services/core-api") });
  },
  reset() {
    run("docker", ["compose", "down", "-v"]);
    rmSync(join(root, "data"), { recursive: true, force: true });
  }
};

if (!tasks[task]) {
  console.error(`Unknown task: ${task || "<missing>"}`);
  process.exit(2);
}
tasks[task]();
