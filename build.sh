mkdir -p build
go build -mod=mod -o build/nymd ./cmd/nymd
go build -mod=mod -o build/nymcli ./cmd/nymcli