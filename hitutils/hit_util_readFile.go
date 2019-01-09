package util

import (
	"encoding/json"
	"io/ioutil"
	"path"
)

func ReadListenerIp(repoPath string) (string, error) {
	var hitconfig HitConfig
	hitconfigstr, err := ioutil.ReadFile(path.Join(repoPath, ClientFileName))
	if err != nil {
		return "", err
	} else {
		json.Unmarshal(hitconfigstr, &hitconfig)
	}
	return hitconfig.IpfsServerIp, nil
}