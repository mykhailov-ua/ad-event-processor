FROM golang:1.25.12-alpine AS modules
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

FROM golang:1.25.12-alpine AS builder
COPY --from=modules /go/pkg/mod /go/pkg/mod
WORKDIR /src
COPY . .
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
