FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm install

COPY . .
RUN npm run build

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/dist /app/dist

CMD ["echo", "Frontend built"]
