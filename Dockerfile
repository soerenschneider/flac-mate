FROM golang:1.26.0 AS build

WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./

ARG VERSION=dev
ARG COMMIT_HASH
ENV CGO_ENABLED=1

RUN CGO_ENABLED=${CGO_ENABLED} go build -ldflags="-w -X 'main.BuildVersion=${VERSION}' -X 'main.CommitHash=${COMMIT_HASH}' -X 'main.GoVersion=$(go version | awk '{print $3}' | sed 's/^go//')'" -o /flac-mate .

FROM gcr.io/distroless/static AS final

LABEL maintainer="soerenschneider"
USER nonroot:nonroot
COPY --from=build --chown=nonroot:nonroot /flac-mate /flac-mate

ENTRYPOINT ["/flac-mate"]
