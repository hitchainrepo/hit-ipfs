package util

import (
	"fmt"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"net"
)

func SendThingsToServerListener(ip_port string, content string) bool {
	conn, err := net.Dial("tcp", ip_port)
	if err != nil {
		fmt.Println("failed to connect to server:", err.Error())
		return false
	}
	conn.Write([]byte(content))
	var response = make([]byte, 1024)
	var count = 0
	for {
		count, err = conn.Read(response)
		if err != nil {
			return false
		} else {
			if string(response[0:count]) == "success" {
				return true
			} else {
				return false
			}
		}
	}
}

func HitListenerAdd(serverIp string, serverPort string, peerId string) error {
	var savedOrNot = SendThingsToServerListener(serverIp+":"+serverPort, "PeerId:"+peerId)
	if savedOrNot == false{
		return errors.New("not initialized")
	} else {
		return nil
	}
}