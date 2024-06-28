unset GOOS
unset GOARCH
unset GOARM
unset CGO_ENABLED
export GOOS=darwin
export GOARCH=amd64
# export GOARM=7
export CGO_ENABLED=1;  
go build -o ../../build/macos/amd64/ -ldflags "-s -w" ../