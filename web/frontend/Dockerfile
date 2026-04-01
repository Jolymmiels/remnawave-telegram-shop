FROM node:24-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm install --include=optional

COPY . .

RUN npm run build
RUN npm prune --omit=dev

FROM node:24-alpine

WORKDIR /app

ENV NODE_ENV=production

COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/build ./build

EXPOSE 4321

CMD ["npm", "run", "start"]
