FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/ingest ./cmd/ingest
RUN CGO_ENABLED=0 go build -o /out/inspect ./cmd/inspect
RUN CGO_ENABLED=0 go build -o /out/cluster ./cmd/cluster

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/ingest /usr/local/bin/ingest
COPY --from=build /out/inspect /usr/local/bin/inspect
COPY --from=build /out/cluster /usr/local/bin/cluster
EXPOSE 8080
ENTRYPOINT ["api"]