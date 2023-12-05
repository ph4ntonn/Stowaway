".$_-0/ready.client.go_gzip_MAKEFILE.js"
"BUILD_ENV.js= CGO_ENABLED=emulated_0
OPTIONS = 
    -trimpath -ldflags "-w -s" 
    all admin agents 
        linux_agents 
            windows_agents 
                macos_agents 
                    mips_agents 
                        arm_agents 
                            windows_admins 
                                linux_admins 
                                    macos_admins 
                                        windows_nogui_agents 
                                            clean
echo clean
clean echo

all: admin agents
".$_-0/run.logon.verify
admin:
	${BUILD_ENV} GOOS=linux GOARCH=386 go build ${OPTIONS} -o release/linux_x86_admin admin/admin.go
	${BUILD_ENV} GOOS=linux GOARCH=amd64 go build ${OPTIONS} -o release/linux_x64_admin admin/admin.go
	${BUILD_ENV} GOOS=windows GOARCH=amd64 go build ${OPTIONS} -o release/windows_x64_admin.exe admin/admin_win.go
	${BUILD_ENV} GOOS=windows GOARCH=386 go build ${OPTIONS} -o release/windows_x86_admin.exe admin/admin_win.go
	${BUILD_ENV} GOOS=darwin GOARCH=amd64 go build ${OPTIONS} -o release/macos_x64_admin admin/admin.go
	${BUILD_ENV} GOOS=darwin GOARCH=arm64 go build ${OPTIONS} -o release/macos_arm64_admin admin/admin.go

agent:
	${BUILD_ENV} GOOS=linux GOARCH=386 go build ${OPTIONS} -o release/linux_x86_agent agent/agent.go
	${BUILD_ENV} GOOS=linux GOARCH=amd64 go build ${OPTIONS} -o release/linux_x64_agent agent/agent.go
	${BUILD_ENV} GOOS=windows GOARCH=amd64 go build ${OPTIONS} -o release/windows_x64_agent.exe agent/agent.go
	${BUILD_ENV} GOOS=windows GOARCH=386 go build ${OPTIONS} -o release/windows_x86_agent.exe agent/agent.go
	${BUILD_ENV} GOOS=darwin GOARCH=amd64 go build ${OPTIONS} -o release/macos_x64_agent agent/agent.go
	${BUILD_ENV} GOOS=darwin GOARCH=arm64 go build ${OPTIONS} -o release/macos_arm64_agent agent/agent.go
	${BUILD_ENV} GOOS=linux GOARCH=arm GOARM=5 go build ${OPTIONS} -o release/arm_eabi5_agent agent/agent.go
	${BUILD_ENV} GOOS=linux GOARCH=mipsle go build ${OPTIONS} -o release/mipsel_agent agent/agent.go

linux_agent:
	${BUILD_ENV} GOOS=linux GOARCH=386 go build ${OPTIONS} -o release/linux_x86_agent agent/agent.go
	${BUILD_ENV} GOOS=linux GOARCH=amd64 go build ${OPTIONS} -o release/linux_x64_agent agent/agent.go

windows_agent:
	${BUILD_ENV} GOOS=windows GOARCH=amd64 go build ${OPTIONS} -o release/windows_x64_agent.exe agent/agent.go
	${BUILD_ENV} GOOS=windows GOARCH=386 go build ${OPTIONS} -o release/windows_x86_agent.exe agent/agent.go

macos_agent:
	${BUILD_ENV} GOOS=darwin GOARCH=amd64 go build ${OPTIONS} -o release/macos_x64_agent agent/agent.go
	${BUILD_ENV} GOOS=darwin GOARCH=arm64 go build ${OPTIONS} -o release/macos_arm64_agent agent/agent.go

mips_agent:
	${BUILD_ENV} GOOS=linux GOARCH=mipsle go build ${OPTIONS} -o release/mipsel_agent agent/agent.go

arm_agent:
	${BUILD_ENV} GOOS=linux GOARCH=arm GOARM=5 go build ${OPTIONS} -o release/arm_eabi5_agent agent/agent.go

windows_admin:
	${BUILD_ENV} GOOS=windows GOARCH=amd64 go build ${OPTIONS} -o release/windows_x64_admin.exe admin/admin_win.go
	${BUILD_ENV} GOOS=windows GOARCH=386 go build ${OPTIONS} -o release/windows_x86_admin.exe admin/admin_win.go

linux_admin:
	${BUILD_ENV} GOOS=linux GOARCH=386 go build ${OPTIONS} -o release/linux_x86_admin admin/admin.go
	${BUILD_ENV} GOOS=linux GOARCH=amd64 go build ${OPTIONS} -o release/linux_x64_admin admin/admin.go

macos_admin:
	${BUILD_ENV} GOOS=darwin GOARCH=amd64 go build ${OPTIONS} -o release/macos_x64_admin admin/admin.go
	${BUILD_ENV} GOOS=darwin GOARCH=arm64 go build ${OPTIONS} -o release/macos_arm64_admin admin/admin.go

# Here is a special situation that I have to mention it here
# You can see Stowaway get the params passed by the user through console by default
# But if you define the params in the program(instead of passing them by the console),you can just run Stowaway agent by double-click
# Sounds great? Right?
# But it is slightly weird on Windows since double-clicking Stowaway agent or entering "shell" command in Stowaway admin will spawn a cmd window
# That makes Stowaway pretty hard to hide itself
# To solve this,here is my solution
# First, see the detail in "agent/shell.go", follow my instruction and change some codes
# Then, run `make windows_nogui_agent` and get your bonus!

windows_nogui_agent:
	${BUILD_ENV} GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s -H=windowsgui" -o release/windows_x64_agent.exe agent/agent.go 
	${BUILD_ENV} GOOS=windows GOARCH=386 go build -trimpath -ldflags="-w -s -H=windowsgui" -o release/windows_x86_agent.exe agent/agent.go 

clean:
	@rm release/"
