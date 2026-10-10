# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/axon ./cmd/axon

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/axon /usr/local/bin/axon
EXPOSE 8080
ENTRYPOINT ["axon"]
CMD ["serve"]
