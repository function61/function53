FROM alpine:latest

# NOTE: because of these args, if you want to build this manually you've to add
#       e.g. --build-arg TARGETARCH=amd64 to $ docker build ...

# "amd64" | "arm" | ...
ARG TARGETARCH
# usually empty. for "linux/arm/v7" => "v7"
ARG TARGETVARIANT

ENTRYPOINT ["function53"]

WORKDIR /function53

CMD ["run"]

COPY "rel/function53_linux-$TARGETARCH" /bin/function53

# the 'write-default-config' writes /function53/config.json file with some reasonable defaults, so
# the image should be usable as-is
RUN apk add --update ca-certificates \
	&& function53 write-default-config
