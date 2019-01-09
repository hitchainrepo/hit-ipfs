package main

import (
	"encoding/json"
	"github.com/ipfs/go-ipfs/core/commands"
	"io/ioutil"
	"path"
)

func readListenerIp(repoPath string) (string, error) {
	var hitconfig commands.HitConfig
	hitconfigstr, err := ioutil.ReadFile(path.Join(repoPath, commands.ClientFileName))
	if err != nil {
		return "", err
	} else {
		json.Unmarshal(hitconfigstr, &hitconfig)
	}
	return hitconfig.IpfsServerIp, nil
}