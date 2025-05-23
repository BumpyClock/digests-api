echo "Building MacOS ARM"

unset GOOS
unset GOARCH
unset GOARM
unset CGO_ENABLED
export GOOS=darwin
export GOARCH=arm64
# export GOARM=7
export CGO_ENABLED=1;  
go build -o ../../build/macos/arm64/ -ldflags "-s -w" ../