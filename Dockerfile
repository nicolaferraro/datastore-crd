FROM golang:1.9.3
WORKDIR /go/src/nicolaferraro.me/datastore-crd
COPY . .
RUN go build
CMD ["./datastore-crd", "-logtostderr"]
