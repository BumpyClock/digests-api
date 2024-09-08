echo "Building Windows ARM"

unset GOOS
unset GOARCH
unset GOARM
unset CGO_ENABLED
export GOOS=windows
export GOARCH=arm64
# export GOARM=7
export CGO_ENABLED=1;  
go build -o ../../build/windows/arm64/ -ldflags "-s -w" -buildmode=pie ../