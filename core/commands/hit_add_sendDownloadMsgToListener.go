package main

import (
	"fmt"
	"github.com/ipfs/go-ipfs/core/commands"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
)

func HitListenerDownload(repoPath string, lastHash string, peerId string) error {
	fmt.Print("in download function")
	serverIp, err := readListenerIp(repoPath)
	if err != nil {
		return errors.New("Error reading hit config!")
	}
	result := SendThingsToServerListener(serverIp + ":" + commands.ServerListenerPort, "Add:"+lastHash+"_"+peerId)
	if result != true {
		return errors.New("Error sending things to server listener!")
	}
	return nil
}