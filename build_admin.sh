echo "Building Stowaway(admin)....."

CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -trimpath -ldflags="-w -s" -o release/linux_x86_admin admin/admin.go
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o release/linux_x64_admin admin/admin.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o release/windows_x64_admin.exe admin/admin.go
CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -trimpath -ldflags="-w -s" -o release/windows_x86_admin.exe admin/admin.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o release/macos_admin admin/admin.go