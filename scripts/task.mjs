import { spawnSync } from "node:child_process";
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

const tasks = {
  setup() {
    if (!existsSync(join(root, ".env"))) copyFileSync(join(root, ".env.example"), join(root, ".env"));
    run("pnpm", ["install", "--frozen-lockfile=false"]);
    run("go", ["mod", "download"], { cwd: join(root, "services/core-api") });
  },
  migrate() {
    run("go", ["run", "./cmd/migrate", "up"], { cwd: join(root, "services/core-api") });
  },
  seed() {
    run("go", ["run", "./cmd/migrate", "seed"], { cwd: join(root, "services/core-api") });
  },
  dev() {
    run("docker", ["compose", "up", "-d", "postgres", "redis", "minio"]);
    run("pnpm", ["--parallel", "--filter", "@opportunity-os/admin-web", "--filter", "@opportunity-os/operator-console", "dev"]);
  },
  test() {
    run("docker", ["compose", "up", "-d", "postgres"]);
    run("go", ["run", "./cmd/migrate", "up"], { cwd: join(root, "services/core-api") });
    run("go", ["test", "./..."], { cwd: join(root, "services/core-api") });
    run("pnpm", ["-r", "--if-present", "test"]);
  },
  lint() {
    run("go", ["vet", "./..."], { cwd: join(root, "services/core-api") });
    run("pnpm", ["-r", "--if-present", "typecheck"]);
  },
  build() {
    run("go", ["build", "./..."], { cwd: join(root, "services/core-api") });
    run("pnpm", ["-r", "--if-present", "build"]);
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
