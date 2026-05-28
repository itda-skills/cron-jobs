FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cron-jobs ./cmd/cron-jobs

FROM alpine:3.20

RUN apk add --no-cache bash ca-certificates curl tzdata \
  && addgroup -S app \
  && adduser -S -G app -u 10001 app \
  && mkdir -p /data/logs /data/scripts/jobs /tmp \
  && chown -R app:app /data /tmp

COPY --from=build /out/cron-jobs /usr/local/bin/cron-jobs

USER app
EXPOSE 8080

ENV APP_ADDR=:8080 \
  APP_DATA_DIR=/data \
  APP_CONFIG_PATH=/data/config.json \
  APP_LOG_DIR=/data/logs \
  APP_SCRIPT_DIR=/data/scripts/jobs \
  APP_TIMEZONE=Asia/Seoul

ENTRYPOINT ["/usr/local/bin/cron-jobs"]
