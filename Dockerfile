FROM node:24-alpine AS ui-build
WORKDIR /ui
COPY ui-app/package.json ui-app/package-lock.json ./
RUN npm ci
COPY ui-app/index.html ui-app/vite.config.js ./
COPY ui-app/src ./src
RUN VITE_API_URL=/ npm run build

FROM scratch AS workspace-assets
COPY --from=ui-build /ui/dist/ /

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
COPY --from=workspace-assets / /usr/share/agent-factory/ui/
ENV FACTORY_UI_DIR=/usr/share/agent-factory/ui
