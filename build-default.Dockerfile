FROM fn61/buildkit-golang:20181204_1302_5eedb86addc826e7

WORKDIR /go/src/github.com/function61/function53

CMD bin/build.sh
