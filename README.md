# Setup
go get labix.org/v2/mgo
go get code.google.com/p/go.net/html

# Building
export GOPATH=$PWD
go run src/ingest.go -d data/test/
