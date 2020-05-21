package main

import (
	"runtime"

	"Stowaway/agent"
)

//注意！需要编译agent模式的程序时，在18行前加上注释符号‘//’，去掉20行的注释符号, 并将上方import Stowaway/admin改为Stowaway/agent
//注意！需要编译admin模式的程序时，在20行前加上注释符号‘//’，去掉18行的注释符号, 并将上方import Stowaway/agent改为Stowaway/admin
//最后执行go build -trimpath -ldflags="-w -s" 命令即可得到对应程序

//Be Mentioned!If you want to compile the agent mode Stowaway,delete the ‘//’ in front of line 20,and add '//' in front of line 18.And change the "import Stowaway/admin" to "import Stowaway/agent"
//Be Mentioned!If you want to compile the admin mode Stowaway,delete the ‘//’ in front of line 18,and add '//' in front of line 20.And change the "import Stowaway/agent" to "import Stowaway/admin"
//Then run go build -trimpath -ldflags="-w -s" command and get result
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// admin.ParseCommand()

	agent.ParseCommand()

}
