FROM golang:1.26-alpine AS build
WORKDIR /src
COPY services/core-api/go.mod services/core-api/go.sum* ./
RUN go mod download
COPY services/core-api/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/core-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/core-api /core-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/core-api"]

