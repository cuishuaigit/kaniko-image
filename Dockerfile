FROM golang:alpine3.14 AS builder
WORKDIR /app/
COPY . /app/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s"  -o /bin/kaniko-image main.go


FROM scratch
COPY --from=builder /bin/kaniko-image /bin/kaniko-image
CMD ["/bin/kaniko-image"]
