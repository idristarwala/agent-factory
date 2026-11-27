FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN sh scripts/build-binaries.sh /out

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates
COPY LICENSE /usr/share/licenses/agent-factory/LICENSE

FROM runtime AS worker
COPY --from=build /out/agent-factory /usr/local/bin/agent-factory
ENTRYPOINT ["agent-factory"]

FROM runtime AS connector
COPY --from=build /out/factory /usr/local/bin/factory
ENTRYPOINT ["factory", "tools", "sample"]

FROM worker AS engine
RUN apk add --no-cache poppler-utils
COPY --from=build /out/factory /usr/local/bin/factory
COPY --from=build /out/api /usr/local/bin/api
