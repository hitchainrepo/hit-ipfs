package util

const (
	// add by Nigel start: declare global variables which can be used by other packages
	ClientFileName = "Hit/config"
	ServerListenerPort = "30004"
	// add by Nigel end
)

type HitConfig struct {
	IpfsServerIp string // selected server ip
	Version string // insert the client version for version management
}
func (p *HitConfig) Init(ipfsServerIp string, version string) {
	p.IpfsServerIp = ipfsServerIp
	p.Version = version
}