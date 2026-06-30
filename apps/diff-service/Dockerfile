FROM oven/bun:1.1-alpine AS builder

WORKDIR /app

COPY package.json ./
COPY tsconfig.json ./

RUN bun install --frozen-lockfile

COPY src ./src

RUN bun run build

FROM oven/bun:1.1-alpine

WORKDIR /app

COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY package.json ./

EXPOSE 5001

CMD ["bun", "run", "start"]