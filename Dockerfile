FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/agent-factory ./cmd/goal \
 && CGO_ENABLED=0 go build -o /out/factory ./cmd/factory \
 && CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/agent-factory /usr/local/bin/agent-factory
COPY --from=build /out/factory /usr/local/bin/factory
COPY --from=build /out/api /usr/local/bin/api
ENTRYPOINT ["agent-factory"]
