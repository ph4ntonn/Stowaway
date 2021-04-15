package main

// import "fmt"

// // /*
// //  * @Author: ph4ntom
// //  * @Date: 2021-03-12 12:41:19
// //  * @LastEditors: ph4ntom
// //  * @LastEditTime: 2021-03-12 12:41:30
// //  */
// // package main

// // import "github.com/xtaci/kcp-go"

// // func main() {
// // 	kcpconn, err := kcp.DialWithOptions("localhost:10000", nil, 10, 3)
// // 	if err != nil {
// // 		panic(err)
// // 	}

// // 	kcpconn.Write([]byte("hello kcp.emmmmmmmmmmmmmmm"))
// // 	select {}
// // }
// func main() {
// 	aa := make(map[int]string)
// 	aa[2] = "b"
// 	aa[1] = "a"
// 	for seq, char := range aa {
// 		fmt.Println("char is ", char, "seq is ", seq)
// 		aa = make(map[int]string)
// 	}

// 	var test1 string
// 	fmt.Println("before ", &test1)
// 	test1 = test1 + "111"
// 	fmt.Println("af	 ", &test1)
// }

import (
	"fmt"

	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
)

func main() {
	add()
	low()
	event()
}

func add() {
	fmt.Println("--- Please press ctrl + shift + q to stop hook ---")
	robotgo.EventHook(hook.KeyDown, []string{"q", "ctrl", "shift"}, func(e hook.Event) {
		fmt.Println("ctrl-shift-q")
		robotgo.EventEnd()
	})

	fmt.Println("--- Please press w---")
	robotgo.EventHook(hook.KeyDown, []string{"w"}, func(e hook.Event) {
		fmt.Println("w")
	})

	s := robotgo.EventStart()
	<-robotgo.EventProcess(s)
}

func low() {
	EvChan := hook.Start()
	defer hook.End()

	for ev := range EvChan {
		fmt.Println("hook: ", ev)
	}
}

func event() {

	str := "I am lilei"

	//string 转[]byte
	b := []byte(str)

	//[]byte转string
	str = string(b)

	//string 转 rune
	r := []rune(str)

	//rune 转 string
	str = string(r)
	ok := robotgo.AddEvents("q", "ctrl", "shift")
	if ok {
		fmt.Println("add events...")
	}

	keve := robotgo.AddEvent("k")
	if keve {
		fmt.Println("you press... ", "k")
	}

	mleft := robotgo.AddEvent("mleft")
	if mleft {
		fmt.Println("you press... ", "mouse left button")
	}
}
