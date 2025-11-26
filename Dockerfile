FROM --platform=$BUILDPLATFORM golang:1.25.3-alpine AS modules
WORKDIR /modules
COPY go.mod go.sum ./
RUN go mod download

FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /frontend
COPY tg-admin/package.json tg-admin/package-lock.json ./
RUN npm ci
COPY tg-admin/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.25.3-alpine AS builder
WORKDIR /app

COPY --from=modules /go/pkg /go/pkg

COPY . .

RUN apk update && apk add --no-cache ca-certificates tzdata
RUN update-ca-certificates

ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG COMMIT=none

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev} -X main.Commit=${COMMIT:-none} -X main.BuildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
    -o /bin/app ./cmd/app

FROM scratch

ARG VERSION
ARG COMMIT

LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT}"
LABEL org.opencontainers.image.source="https://github.com/${GITHUB_REPOSITORY}"
LABEL org.opencontainers.image.description="Remnawave Telegram Shop Bot"
LABEL org.opencontainers.image.licenses="MIT"

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /bin/app /app/app

COPY --from=builder /app/db /db
COPY --from=builder /app/translations /translations
COPY --from=frontend /frontend/dist /tg-admin/dist

USER 1000

ENV DISABLE_ENV_FILE=true

CMD ["/app/app"]