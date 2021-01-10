echo "Building Stowaway(agent)....."

CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -trimpath -ldflags="-w -s" -o linux_x86_agent
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o linux_x64_agent
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o windows_x64_agent.exe
CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -trimpath -ldflags="-w -s" -o windows_x86_agent.exe
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o macos_agent
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -trimpath -ldflags="-w -s" -o arm_eabi5_agent
CGO_ENABLED=0 GOOS=linux GOARCH=mipsle go build -trimpath -ldflags="-w -s" -o mipsel_agent

# Here is a special situation that i have to mention it here
# You can see Stowaway get the params passed by the user through console by default
# But if you define the params in the program(instead of passing them by the console),you can just run Stowaway agent by double-click
# Sounds great? Right?
# But it is slightly weird on Windows since double-clicking Stowaway agent or entering "shell" command in Stowaway admin will spawn a cmd window
# That makes Stowaway pretty hard to hide itself
# To solve this,here is my solution
# First, see the detail in "agent/shell.go", follow my instruction and change some codes
# Then,compile Stowaway(Windows platform) solo by using the following two sentences and get your bonus!

#CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s -H=windowsgui" -o windows_x64_agent.exe
#CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -trimpath -ldflags="-w -s -H=windowsgui" -o windows_x86_agent.exe