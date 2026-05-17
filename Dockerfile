FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ocp-node .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/ocp-node .
EXPOSE 5000
CMD ["./ocp-node"]
