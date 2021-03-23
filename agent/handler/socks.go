/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 18:57:46
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-23 19:26:21
 */
package handler

type Socks struct {
	Username string
	Password string
}

func NewSocks(username, password string) *Socks {
	socks := new(Socks)
	socks.Username = username
	socks.Password = password
	return socks
}

func (socks *Socks) Start() {

}
