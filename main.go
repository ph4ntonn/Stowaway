package main

import (
	"Stowaway/agent"
	"log"
	"os"
	"runtime"

	"github.com/urfave/cli/v2"
)

const version = "1.5"

//注意！需要编译agent模式的程序时，在20行的admin.Command前加上注释符号‘//’，去掉19行的注释符号, 并将上方import Stowaway/admin改为Stowaway/agent
//注意！需要编译admin模式的程序时，在19行的agent.Command前加上注释符号‘//’，去掉20行的注释符号, 并将上方import Stowaway/agent改为Stowaway/admin
//最后执行go build -ldflags="-w -s" 命令即可得到对应程序

//Be Mentioned!If you want to compile the agent mode Stowaway,delete the ‘//’ in front of agent.Command,and add '//' in front of admin.Command.And change the "import Stowaway/admin" to "import Stowaway/agent"
//Be Mentioned!If you want to compile the admin mode Stowaway,delete the ‘//’ in front of admin.Command,and add '//' in front of agent.Command.And change the "import Stowaway/agent" to "import Stowaway/admin"
//Then run go build -ldflags="-w -s" command and get result
func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	app := &cli.App{}
	cli.AppHelpTemplate = ``
	app.Commands = []*cli.Command{
		agent.Command,
		//admin.Command,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal("[*]", err)
	}
}
