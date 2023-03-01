FROM alpine:latest

WORKDIR /app
ADD ./db /app/db
ADD ./bin/integritySum /app

ENTRYPOINT ["/app/integritySum"]
