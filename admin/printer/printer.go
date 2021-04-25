package printer

import "github.com/fatih/color"

var (
	Warning func(format string, a ...interface{})
	Success func(format string, a ...interface{})
	Fail    func(format string, a ...interface{})
)

func InitPrinter() {
	Warning = color.New(color.FgYellow).PrintfFunc()
	Success = color.New(color.FgGreen).PrintfFunc()
	Fail = color.New(color.FgRed).PrintfFunc()
}
