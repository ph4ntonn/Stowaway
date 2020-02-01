echo "Building Stowaway(agent)....."

CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags="-w -s" -o linux_x86_agent
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o linux_x64_agent
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o windows_x64_agent.exe
CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags="-w -s" -o windows_x86_agent.exe
go build -ldflags="-w -s" -o macos_agent
