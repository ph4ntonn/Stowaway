package main

import (
	"Stowaway/agent"
	"log"
	"os"
	"runtime"

	"github.com/urfave/cli/v2"
)

//注意！需要编译agent模式的程序时，在29行及30行前加上注释符号‘//’，去掉26行及27行的注释符号, 并将上方import Stowaway/admin改为Stowaway/agent
//注意！需要编译admin模式的程序时，在26行及27行前加上注释符号‘//’，去掉29行及30行的注释符号, 并将上方import Stowaway/agent改为Stowaway/admin
//最后执行go build -trimpath -ldflags="-w -s" 命令即可得到对应程序

//Be Mentioned!If you want to compile the agent mode Stowaway,delete the ‘//’ in front of 26 and 27 lines,and add '//' in front of 29 and 30 lines.And change the "import Stowaway/admin" to "import Stowaway/agent"
//Be Mentioned!If you want to compile the admin mode Stowaway,delete the ‘//’ in front of 29 and 30 lines,and add '//' in front of 26 and 27 lines.And change the "import Stowaway/agent" to "import Stowaway/admin"
//Then run go build -trimpath -ldflags="-w -s" command and get result
func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	app := &cli.App{}
	cli.AppHelpTemplate = ``

	app.Flags = agent.Flags
	app.Action = agent.Action

	// app.Flags = admin.Flags
	// app.Action = admin.Action

	app.UseShortOptionHandling = true
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal("[*]", err)
	}
}
