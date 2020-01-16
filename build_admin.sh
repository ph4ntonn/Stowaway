echo "Building Stowaway(admin)....."

CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags="-w -s" -o linux_x86_admin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o linux_x64_admin
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o windows_x64_admin.exe
go build -ldflags="-w -s" -o macos_admin
