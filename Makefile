.PHONY: setup migrate seed dev test lint build e2e production-check reset

setup:
	pnpm node scripts/task.mjs setup
migrate:
	pnpm node scripts/task.mjs migrate
seed:
	pnpm node scripts/task.mjs seed
dev:
	pnpm node scripts/task.mjs dev
test:
	pnpm node scripts/task.mjs test
lint:
	pnpm node scripts/task.mjs lint
build:
	pnpm node scripts/task.mjs build
e2e:
	pnpm node scripts/task.mjs e2e
production-check:
	pnpm node scripts/task.mjs productionCheck
reset:
	pnpm node scripts/task.mjs reset
