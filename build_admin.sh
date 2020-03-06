echo "Building Stowaway(admin)....."

CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -trimpath -ldflags="-w -s" -o linux_x86_admin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o linux_x64_admin
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o windows_x64_admin.exe
CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -trimpath -ldflags="-w -s" -o windows_x86_admin.exe
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o macos_admin
