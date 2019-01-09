package commands

import (
	"github.com/ipfs/go-ipfs/cmd/ipfs/util"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
)

func HitListenerDownload(repoPath string, lastHash string, peerId string) error {
	serverIp, err := util.ReadListenerIp(repoPath)
	if err != nil {
		return errors.New("Error reading hit config!")
	}
	result := util.SendThingsToServerListener(serverIp + ":" + ServerListenerPort, "Add:"+lastHash+"_"+peerId)
	if result != true {
		return errors.New("Error sending things to server listener!")
	}
	return nil
}