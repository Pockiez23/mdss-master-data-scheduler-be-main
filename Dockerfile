ARG IMAGE_PREFIX
ARG PROXY_IMAGE_PREFIX
FROM ${IMAGE_PREFIX}/base:stable AS builder

# Change the following line according to your project structure
# USER for runtime must be number (Default: 65532)
ARG APP_NAME=server

USER root

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app cmd/${APP_NAME}/main.go

#####################

FROM ${PROXY_IMAGE_PREFIX}/alpine:3.21.2

RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

COPY --from=builder /usr/src/app/app /usr/local/bin/app

RUN apk add --no-cache libcap && \
    setcap 'cap_net_bind_service=+ep' /usr/local/bin/app

USER 10001

CMD ["/usr/local/bin/app"]
