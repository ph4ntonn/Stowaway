package main

import "fmt"

// /*
//  * @Author: ph4ntom
//  * @Date: 2021-03-12 12:41:19
//  * @LastEditors: ph4ntom
//  * @LastEditTime: 2021-03-12 12:41:30
//  */
// package main

// import "github.com/xtaci/kcp-go"

// func main() {
// 	kcpconn, err := kcp.DialWithOptions("localhost:10000", nil, 10, 3)
// 	if err != nil {
// 		panic(err)
// 	}

// 	kcpconn.Write([]byte("hello kcp.emmmmmmmmmmmmmmm"))
// 	select {}
// }
func main() {
	aa := make(map[int]string)
	aa[2] = "b"
	aa[1] = "a"
	for seq, char := range aa {
		fmt.Println("char is ", char, "seq is ", seq)
		aa = make(map[int]string)
	}

	var test1 string
	fmt.Println("before ", &test1)
	test1 = test1 + "111"
	fmt.Println("af	 ", &test1)
}
