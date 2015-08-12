# Setup
apt-get install bzr mercurial
export GOPATH=$PWD
go get labix.org/v2/mgo
go get code.google.com/p/go.net/html

# Building
go run src/ingest.go -d data/test/
