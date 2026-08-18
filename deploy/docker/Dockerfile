
FROM golang:1.25.12-alpine AS modules
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

FROM node:22-alpine AS web
WORKDIR /src
COPY web/ ./web/
RUN node web/scripts/build.mjs

FROM golang:1.25.12-alpine AS builder
ARG SKIP_CODEGEN=0
RUN apk add --no-cache bash
COPY --from=modules /go/pkg/mod /go/pkg/mod
WORKDIR /src
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN if [ "$SKIP_CODEGEN" = "1" ]; then echo "skip codegen (pre-generated tree)"; else bash scripts/ci/gen.sh --proto; fi
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/tracker ./cmd/tracker
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/processor ./cmd/processor
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/ivt-detector ./cmd/ivt-detector
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/fraud-scorer ./cmd/fraud-scorer
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/broker ./cmd/broker
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/region-proxy ./cmd/region-proxy
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/log-shipper ./cmd/log-shipper
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/control ./cmd/control
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -ldflags="-s -w" -o /bin/alertmanager-telegram ./cmd/alertmanager-telegram

FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/tracker /tracker
COPY --from=builder /bin/processor /processor
COPY --from=builder /bin/ivt-detector /ivt-detector
COPY --from=builder /bin/fraud-scorer /fraud-scorer
COPY --from=builder /bin/broker /broker
COPY --from=builder /bin/region-proxy /region-proxy
COPY --from=builder /bin/log-shipper /log-shipper
COPY --from=builder /bin/control /control
COPY --from=builder /bin/alertmanager-telegram /alertmanager-telegram
USER nonroot:nonroot
ENTRYPOINT ["/tracker"]
