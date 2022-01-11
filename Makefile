.PHONY: admin agent

GO_BUILD_PARAMS := -trimpath -ldflags="-s -w"

admin:
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build $(GO_BUILD_PARAMS) -o release/linux_x86_admin admin/admin.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_PARAMS) -o release/linux_x64_admin admin/admin.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GO_BUILD_PARAMS) -o release/windows_x64_admin.exe admin/admin_win.go
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build $(GO_BUILD_PARAMS) -o release/windows_x86_admin.exe admin/admin_win.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_PARAMS) -o release/macos_admin admin/admin.go


agent:
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -trimpath -ldflags="-w -s" -o release/linux_x86_agent agent/agent.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o release/linux_x64_agent agent/agent.go
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o release/windows_x64_agent.exe agent/agent.go
	GOOS=windows GOARCH=386 go build -trimpath -ldflags="-w -s" -o release/windows_x86_agent.exe agent/agent.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o release/macos_agent agent/agent.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -trimpath -ldflags="-w -s" -o release/arm_eabi5_agent agent/agent.go
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle go build -trimpath -ldflags="-w -s" -o release/mipsel_agent agent/agent.go


test:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GO_BUILD_PARAMS) -o release/test_windows_x64_admin.exe admin/admin_win.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GO_BUILD_PARAMS) -o release/test_windows_x64_agent.exe agent/agent.go

