echo "Building Linux ARM"

unset GOOS
unset GOARCH
unset GOARM
unset CGO_ENABLED
export GOOS=linux
export GOARCH=arm64
# export GOARM=7
export CGO_ENABLED=1
go build -o ../../build/linux/arm64/ -ldflags "-s -w" -buildmode=pie ../