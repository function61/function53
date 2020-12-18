FROM alpine:latest

# NOTE: because of these args, if you want to build this manually you've to add
#       e.g. --build-arg TARGETARCH=amd64 to $ docker build ...

# "amd64" | "arm" | ...
ARG TARGETARCH
# usually empty. for "linux/arm/v7" => "v7"
ARG TARGETVARIANT

ENTRYPOINT ["function53"]

CMD ["run"]

RUN apk add --update ca-certificates

COPY "rel/function53_linux-$TARGETARCH" /bin/function53

