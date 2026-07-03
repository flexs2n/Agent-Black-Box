FROM node:20-alpine AS pnpm-base
RUN corepack enable && corepack prepare pnpm@latest --activate

FROM pnpm-base AS diff-engine-builder
WORKDIR /app
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json tsconfig.json ./
COPY packages/diff-engine ./packages/diff-engine
RUN pnpm install --frozen-lockfile --filter @blackbox/diff-engine
RUN pnpm --filter @blackbox/diff-engine run build

FROM pnpm-base AS diff-service-builder
WORKDIR /app
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY packages/diff-engine/package.json ./packages/diff-engine/package.json
COPY --from=diff-engine-builder /app/packages/diff-engine/dist ./packages/diff-engine/dist
COPY --from=diff-engine-builder /app/packages/diff-engine/package.json ./packages/diff-engine/package.json
COPY apps/diff-service ./apps/diff-service
RUN pnpm install --frozen-lockfile --filter @blackbox/diff-service
RUN pnpm --filter @blackbox/diff-service run build

FROM oven/bun:1.1-alpine
RUN apk add --no-cache curl
WORKDIR /app
COPY --from=diff-service-builder /app/node_modules ./node_modules
COPY --from=diff-service-builder /app/apps/diff-service/dist ./apps/diff-service/dist
COPY --from=diff-service-builder /app/apps/diff-service/node_modules ./apps/diff-service/node_modules
COPY --from=diff-service-builder /app/packages/diff-engine/dist ./packages/diff-engine/dist
COPY --from=diff-service-builder /app/packages/diff-engine/package.json ./packages/diff-engine/package.json
COPY apps/diff-service/package.json ./apps/diff-service/package.json
EXPOSE 5001
CMD ["bun", "apps/diff-service/dist/index.js"]
