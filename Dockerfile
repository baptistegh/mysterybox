FROM golang:1.25 AS build

WORKDIR /app
COPY . /app/
RUN go mod download && \
    go generate ./... && \
    go build -o bin/mysterybox -v cmd/main.go

FROM gcr.io/distroless/cc-debian12:nonroot AS base

WORKDIR /app
COPY --from=build /app/bin/mysterybox /usr/local/bin/mysterybox

ENTRYPOINT [ "/usr/local/bin/mysterybox" ]