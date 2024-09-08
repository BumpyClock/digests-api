echo "Building linux amd64"

unset GOOS
unset GOARCH
unset GOARM
unset CGO_ENABLED
export GOOS=linux
export GOARCH=amd64
# export GOARM=7
export CGO_ENABLED=1;  
go build -o ../../build/linux/amd64/ -ldflags "-s -w" ../