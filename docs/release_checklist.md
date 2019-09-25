# Release Checklist

## Prerequisites

* [Go](https://golang.org>) 1.10 or later. Note: older versions might still work but have not been tested.
* [protoc](https://github.com/protocolbuffers/protobuf) version3.

## Release Processs

* ensure local copy of the repository is on master and up-to-date:

```bash
cd $GOPATH/src/github.com/nymtech/nym-validator
git checkout master
git pull origin master
```

* run all tests and ensure they pass:

```bash
cd $GOPATH/src/github.com/nymtech/nym-validator
go test -v ./...
```

* ensure protobuf-generated files did not unexpectedly change:

```bash
cd $GOPATH/src/github.com/nymtech/nym-validator
go generate
git status --porcelain
```

* ensure generated queue files did not unexpectedly change:

```bash
cd $GOPATH/src/github.com/nymtech/nym-validator/crypto/coconut/concurrency/jobqueue
go generate
cd $GOPATH/src/github.com/nymtech/nym-validator/server/comm/requestqueue/
go generate
git status --porcelain
```

* create appropriate version tag according to [Semantic Versioning](https://semver.org/) to reflect extent of changes made:

```bash
cd $GOPATH/src/github.com/nymtech/nym-validator
git tag -s vX.Y.Z
git push origin vX.Y.Z
```

* update changelog.md with appropriate information reflecting changes in the new version.
