echo "Building Windows amd64"

unset GOOS
unset GOARCH
unset GOARM
unset CGO_ENABLED
export GOOS=windows
export GOARCH=amd64
# export GOARM=7
export CGO_ENABLED=1;  
go build -o ../../build/windows/amd64/ -ldflags "-s -w" -buildmode=pie ../