FROM node:20-alpine AS base
RUN corepack enable && corepack prepare pnpm@latest --activate

FROM base AS builder
WORKDIR /app
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json tsconfig.json ./
COPY apps/web ./apps/web
RUN pnpm install --frozen-lockfile --filter web
RUN pnpm --filter web run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/apps/web/.next/standalone/. ./
COPY --from=builder /app/apps/web/.next/static ./.next/static
EXPOSE 3000
ENV PORT=3000
ENV NODE_ENV=production
CMD ["node", "server.js"]