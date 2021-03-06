package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	_ "expvar"
	"fmt"
	utilmain "github.com/ipfs/go-ipfs/cmd/ipfs/util"
	oldcmds "github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/commands"
	"github.com/ipfs/go-ipfs/core/commands/cmdenv"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/core/corerepo"
	nodeMount "github.com/ipfs/go-ipfs/fuse/node"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	migrate "github.com/ipfs/go-ipfs/repo/fsrepo/migrations"
	hitutil "github.com/ipfs/go-ipfs/hitutils"
	"github.com/robfig/cron"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gx/ipfs/QmPTfgFTo9PFr1PvPKyKoeMgBvYPh6cX3aDP7DHKVbnCbi/go-ipfs-cmds"
	"gx/ipfs/QmSP88ryZkHSRn1fnngAaV2Vcn63WUJzAavnRM9CVdU1Ky/go-ipfs-cmdkit"
	mprome "gx/ipfs/QmUHHsirrDtP6WEHhE8SZeG672CLqDJn6XGzAHnvBHUiA3/go-metrics-prometheus"
	"gx/ipfs/QmV6FjemM1K8oXjrvuq3wuVWWoU2TLDPmNnKrxHzY3v6Ai/go-multiaddr-net"
	"gx/ipfs/QmYYv3QFnfQbiwmi1tpkgKF8o4xFnZoBrvpupTiGJwL9nH/client_golang/prometheus"
	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"

	ci "gx/ipfs/QmPvyPwuCgJ7pDmrKDxRtsScJgBaM5h4EpRL2qQJsmXf4n/go-libp2p-crypto"
)

const (
	adjustFDLimitKwd          = "manage-fdlimit"
	enableGCKwd               = "enable-gc"
	initOptionKwd             = "init"
	initProfileOptionKwd      = "init-profile"
	ipfsMountKwd              = "mount-ipfs"
	ipnsMountKwd              = "mount-ipns"
	migrateKwd                = "migrate"
	mountKwd                  = "mount"
	offlineKwd                = "offline"
	routingOptionKwd          = "routing"
	routingOptionSupernodeKwd = "supernode"
	routingOptionDHTClientKwd = "dhtclient"
	routingOptionDHTKwd       = "dht"
	routingOptionNoneKwd      = "none"
	routingOptionDefaultKwd   = "default"
	unencryptTransportKwd     = "disable-transport-encryption"
	unrestrictedApiAccessKwd  = "unrestricted-api"
	writableKwd               = "writable"
	enableFloodSubKwd         = "enable-pubsub-experiment"
	enableIPNSPubSubKwd       = "enable-namesys-pubsub"
	enableMultiplexKwd        = "enable-mplex-experiment"
	// apiAddrKwd    = "address-api"
	// swarmAddrKwd  = "address-swarm"
)

var daemonCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Run a network-connected IPFS node.",
		ShortDescription: `
'ipfs daemon' runs a persistent ipfs daemon that can serve commands
over the network. Most applications that use IPFS will do so by
communicating with a daemon over the HTTP API. While the daemon is
running, calls to 'ipfs' commands will be sent over the network to
the daemon.
`,
		LongDescription: `
The daemon will start listening on ports on the network, which are
documented in (and can be modified through) 'ipfs config Addresses'.
For example, to change the 'Gateway' port:

  ipfs config Addresses.Gateway /ip4/127.0.0.1/tcp/8082

The API address can be changed the same way:

  ipfs config Addresses.API /ip4/127.0.0.1/tcp/5002

Make sure to restart the daemon after changing addresses.

By default, the gateway is only accessible locally. To expose it to
other computers in the network, use 0.0.0.0 as the ip address:

  ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/8080

Be careful if you expose the API. It is a security risk, as anyone could
control your node remotely. If you need to control the node remotely,
make sure to protect the port as you would other services or database
(firewall, authenticated proxy, etc).

HTTP Headers

ipfs supports passing arbitrary headers to the API and Gateway. You can
do this by setting headers on the API.HTTPHeaders and Gateway.HTTPHeaders
keys:

  ipfs config --json API.HTTPHeaders.X-Special-Header '["so special :)"]'
  ipfs config --json Gateway.HTTPHeaders.X-Special-Header '["so special :)"]'

Note that the value of the keys is an _array_ of strings. This is because
headers can have more than one value, and it is convenient to pass through
to other libraries.

CORS Headers (for API)

You can setup CORS headers the same way:

  ipfs config --json API.HTTPHeaders.Access-Control-Allow-Origin '["example.com"]'
  ipfs config --json API.HTTPHeaders.Access-Control-Allow-Methods '["PUT", "GET", "POST"]'
  ipfs config --json API.HTTPHeaders.Access-Control-Allow-Credentials '["true"]'

Shutdown

To shutdown the daemon, send a SIGINT signal to it (e.g. by pressing 'Ctrl-C')
or send a SIGTERM signal to it (e.g. with 'kill'). It may take a while for the
daemon to shutdown gracefully, but it can be killed forcibly by sending a
second signal.

IPFS_PATH environment variable

ipfs uses a repository in the local file system. By default, the repo is
located at ~/.ipfs. To change the repo location, set the $IPFS_PATH
environment variable:

  export IPFS_PATH=/path/to/ipfsrepo

Routing

IPFS by default will use a DHT for content routing. There is a highly
experimental alternative that operates the DHT in a 'client only' mode that
can be enabled by running the daemon as:

  ipfs daemon --routing=dhtclient

This will later be transitioned into a config option once it gets out of the
'experimental' stage.

DEPRECATION NOTICE

Previously, ipfs used an environment variable as seen below:

  export API_ORIGIN="http://localhost:8888/"

This is deprecated. It is still honored in this version, but will be removed
in a future version, along with this notice. Please move to setting the HTTP
Headers.
`,
	},

	Options: []cmdkit.Option{
		cmdkit.BoolOption(initOptionKwd, "Initialize ipfs with default settings if not already initialized"),
		cmdkit.StringOption(initProfileOptionKwd, "Configuration profiles to apply for --init. See ipfs init --help for more"),
		cmdkit.StringOption(routingOptionKwd, "Overrides the routing option").WithDefault(routingOptionDefaultKwd),
		cmdkit.BoolOption(mountKwd, "Mounts IPFS to the filesystem"),
		cmdkit.BoolOption(writableKwd, "Enable writing objects (with POST, PUT and DELETE)"),
		cmdkit.StringOption(ipfsMountKwd, "Path to the mountpoint for IPFS (if using --mount). Defaults to config setting."),
		cmdkit.StringOption(ipnsMountKwd, "Path to the mountpoint for IPNS (if using --mount). Defaults to config setting."),
		cmdkit.BoolOption(unrestrictedApiAccessKwd, "Allow API access to unlisted hashes"),
		cmdkit.BoolOption(unencryptTransportKwd, "Disable transport encryption (for debugging protocols)"),
		cmdkit.BoolOption(enableGCKwd, "Enable automatic periodic repo garbage collection"),
		cmdkit.BoolOption(adjustFDLimitKwd, "Check and raise file descriptor limits if needed").WithDefault(true),
		cmdkit.BoolOption(offlineKwd, "Run offline. Do not connect to the rest of the network but provide local API."),
		cmdkit.BoolOption(migrateKwd, "If true, assume yes at the migrate prompt. If false, assume no."),
		cmdkit.BoolOption(enableFloodSubKwd, "Instantiate the ipfs daemon with the experimental pubsub feature enabled."),
		cmdkit.BoolOption(enableIPNSPubSubKwd, "Enable IPNS record distribution through pubsub; enables pubsub."),
		cmdkit.BoolOption(enableMultiplexKwd, "Add the experimental 'go-multiplex' stream muxer to libp2p on construction.").WithDefault(true),

		// TODO: add way to override addresses. tricky part: updating the config if also --init.
		// cmdkit.StringOption(apiAddrKwd, "Address for the daemon rpc API (overrides config)"),
		// cmdkit.StringOption(swarmAddrKwd, "Address for the swarm socket (overrides config)"),
	},
	Subcommands: map[string]*cmds.Command{},
	Run:         daemonFunc,
}

// defaultMux tells mux to serve path using the default muxer. This is
// mostly useful to hook up things that register in the default muxer,
// and don't provide a convenient http.Handler entry point, such as
// expvar and http/pprof.
func defaultMux(path string) corehttp.ServeOption {
	return func(node *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.Handle(path, http.DefaultServeMux)
		return mux, nil
	}
}

// add by Nigel start: change string to int64 according to the ascii table by adding different letters
func stringToInt64(s string) int64 {
	result := int64(0)
	for _, c := range s {
		result += int64(c)
	}
	return result
}
// add by Nigel end

// add by Nigel start: read local file
func readIpPort(req *cmds.Request) (string, error) {
	repoPath, err := getRepoPath(req)
	if err != nil {
		return "", err
	} else {
		ip_port, err := ioutil.ReadFile(path.Join(repoPath, hitutil.ClientFileName))
		if err != nil {
			return "", err
		} else {
			return string(ip_port), err
		}
	}
}
// add by Nigel end

// add by Nigel start: send restful webservice request
func sendWebServiceRequest(reportRequestItem map[string]interface{}, url string, method string) (map[string]interface{}, error){
	mapResult := make(map[string]interface{})
	bytesData, err := json.Marshal(reportRequestItem)
	if err != nil {
		return mapResult, err
	}
	reader := bytes.NewReader(bytesData)
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		return mapResult, err
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return mapResult, err
	}
	if resp.StatusCode != 200 {
		fmt.Println("Error with the request!")
		return mapResult, errors.New("request the server!")
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return mapResult, err
	}
	if err := json.Unmarshal([]byte(string(respBytes)), &mapResult); err != nil {
		return mapResult, err
	}
	return mapResult, nil
}
// add by Nigel end

// add by Nigel start: rsa encryption
func RsaSignWithSha256Hex(data string, prvKey string) (string, error) {
	keyByts, err := hex.DecodeString(prvKey)
	if err != nil {
		fmt.Println("error line 265")
		fmt.Println(err)
		return "", err
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(keyByts)
	if err != nil {
		fmt.Println("error line 271")
		fmt.Println("ParsePKCS8PrivateKey err", err)
		return "", err
	}
	h := crypto.SHA256.New()
	h.Write([]byte([]byte(data)))
	hash := h.Sum(nil)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey.(*rsa.PrivateKey), crypto.SHA256, hash[:])
	if err != nil {
		fmt.Println("error line 280")
		fmt.Printf("Error from signing: %s\n", err)
		return "", err
	}
	out := hex.EncodeToString(signature)
	return out, nil
}
// add by Nigel end

func daemonFunc(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
	// Inject metrics before we do anything
	err := mprome.Inject()
	if err != nil {
		log.Errorf("Injecting prometheus handler for metrics failed with message: %s\n", err.Error())
	}

	// let the user know we're going.
	fmt.Printf("Initializing daemon...\n")

	managefd, _ := req.Options[adjustFDLimitKwd].(bool)
	if managefd {
		if err := utilmain.ManageFdLimit(); err != nil {
			log.Errorf("setting file descriptor limit: %s", err)
		}
	}

	cctx := env.(*oldcmds.Context)

	go func() {
		<-req.Context.Done()
		fmt.Println("Received interrupt signal, shutting down...")
		fmt.Println("(Hit ctrl-c again to force-shutdown the daemon.)")
	}()

	// check transport encryption flag.
	unencrypted, _ := req.Options[unencryptTransportKwd].(bool)
	if unencrypted {
		log.Warningf(`Running with --%s: All connections are UNENCRYPTED.
		You will not be able to connect to regular encrypted networks.`, unencryptTransportKwd)
	}

	// first, whether user has provided the initialization flag. we may be
	// running in an uninitialized state.
	initialize, _ := req.Options[initOptionKwd].(bool)
	if initialize {

		cfg := cctx.ConfigRoot
		if !fsrepo.IsInitialized(cfg) {
			profiles, _ := req.Options[initProfileOptionKwd].(string)

			err := initWithDefaults(os.Stdout, cfg, profiles)
			if err != nil {
				re.SetError(err, cmdkit.ErrNormal)
				return
			}
		}
	}

	// acquire the repo lock _before_ constructing a node. we need to make
	// sure we are permitted to access the resources (datastore, etc.)
	repo, err := fsrepo.Open(cctx.ConfigRoot)
	switch err {
	default:
		re.SetError(err, cmdkit.ErrNormal)
		return
	case fsrepo.ErrNeedMigration:
		domigrate, found := req.Options[migrateKwd].(bool)
		fmt.Println("Found outdated fs-repo, migrations need to be run.")

		if !found {
			domigrate = YesNoPrompt("Run migrations now? [y/N]")
		}

		if !domigrate {
			fmt.Println("Not running migrations of fs-repo now.")
			fmt.Println("Please get fs-repo-migrations from https://dist.ipfs.io")
			re.SetError(fmt.Errorf("fs-repo requires migration"), cmdkit.ErrNormal)
			return
		}

		err = migrate.RunMigration(fsrepo.RepoVersion)
		if err != nil {
			fmt.Println("The migrations of fs-repo failed:")
			fmt.Printf("  %s\n", err)
			fmt.Println("If you think this is a bug, please file an issue and include this whole log output.")
			fmt.Println("  https://github.com/ipfs/fs-repo-migrations")
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		repo, err = fsrepo.Open(cctx.ConfigRoot)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
	case nil:
		break
	}

	cfg, err := cctx.GetConfig()
	if err != nil {
		re.SetError(err, cmdkit.ErrNormal)
		return
	}

	offline, _ := req.Options[offlineKwd].(bool)
	ipnsps, _ := req.Options[enableIPNSPubSubKwd].(bool)
	pubsub, _ := req.Options[enableFloodSubKwd].(bool)
	mplex, _ := req.Options[enableMultiplexKwd].(bool)

	// Start assembling node config
	ncfg := &core.BuildCfg{
		Repo:      repo,
		Permanent: true, // It is temporary way to signify that node is permanent
		Online:    !offline,
		DisableEncryptedConnections: unencrypted,
		ExtraOpts: map[string]bool{
			"pubsub": pubsub,
			"ipnsps": ipnsps,
			"mplex":  mplex,
		},
		//TODO(Kubuxu): refactor Online vs Offline by adding Permanent vs Ephemeral
	}

	routingOption, _ := req.Options[routingOptionKwd].(string)
	if routingOption == routingOptionDefaultKwd {
		cfg, err := repo.Config()
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		routingOption = cfg.Routing.Type
		if routingOption == "" {
			routingOption = routingOptionDHTKwd
		}
	}
	switch routingOption {
	case routingOptionSupernodeKwd:
		re.SetError(errors.New("supernode routing was never fully implemented and has been removed"), cmdkit.ErrNormal)
		return
	case routingOptionDHTClientKwd:
		ncfg.Routing = core.DHTClientOption
	case routingOptionDHTKwd:
		ncfg.Routing = core.DHTOption
	case routingOptionNoneKwd:
		ncfg.Routing = core.NilRouterOption
	default:
		re.SetError(fmt.Errorf("unrecognized routing option: %s", routingOption), cmdkit.ErrNormal)
		return
	}

	node, err := core.NewNode(req.Context, ncfg)
	if err != nil {
		log.Error("error from node construction: ", err)
		re.SetError(err, cmdkit.ErrNormal)
		return
	}
	node.SetLocal(false)

	if node.PNetFingerprint != nil {
		fmt.Println("Swarm is limited to private network of peers with the swarm key")
		fmt.Printf("Swarm key fingerprint: %x\n", node.PNetFingerprint)
	}

	printSwarmAddrs(node)

	defer func() {
		// We wait for the node to close first, as the node has children
		// that it will wait for before closing, such as the API server.
		node.Close()

		select {
		case <-req.Context.Done():
			log.Info("Gracefully shut down daemon")
		default:
		}
	}()

	cctx.ConstructNode = func() (*core.IpfsNode, error) {
		return node, nil
	}

	// construct api endpoint - every time
	apiErrc, err := serveHTTPApi(req, cctx)
	if err != nil {
		re.SetError(err, cmdkit.ErrNormal)
		return
	}

	// construct fuse mountpoints - if the user provided the --mount flag
	mount, _ := req.Options[mountKwd].(bool)
	if mount && offline {
		re.SetError(errors.New("mount is not currently supported in offline mode"),
			cmdkit.ErrClient)
		return
	}
	if mount {
		if err := mountFuse(req, cctx); err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
	}

	// repo blockstore GC - if --enable-gc flag is present
	gcErrc, err := maybeRunGC(req, node)
	if err != nil {
		re.SetError(err, cmdkit.ErrNormal)
		return
	}

	// construct http gateway - if it is set in the config
	var gwErrc <-chan error
	if len(cfg.Addresses.Gateway) > 0 {
		var err error
		gwErrc, err = serveHTTPGateway(req, cctx)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
	}

	// initialize metrics collector
	prometheus.MustRegister(&corehttp.IpfsNodeCollector{Node: node})

	// add by Nigel start: generate a temporary rsa key pair
	sk, pk, err := ci.GenerateKeyPair(ci.RSA, commands.NBitsForKeypairDefault) // pk represents the public key
	if err != nil {
		re.SetError(err, cmdkit.ErrNormal)
		return
	}

	publicBytes, err := pk.Raw()
	if err != nil {
		re.SetError(err, cmdkit.ErrNormal)
		return
	}
	pubKeyBytes := pem.EncodeToMemory(&pem.Block{
		Bytes: publicBytes,
		Type:  "PUBLIC KEY",
	})

	// read ip and port from local file
	ip_port, err := readIpPort(req)
	if err != nil {
		re.SetError(err, cmdkit.ErrNormal)
		return
	}
	ip_port_tmp := strings.Split(ip_port, ":")
	if len(ip_port_tmp) != 2 {
		fmt.Println("error with the server ip address")
		re.SetError(err, cmdkit.ErrNormal)
		return
	}
	ip := ip_port_tmp[0]

	n, err := cmdenv.GetNode(env) // get nodeId
	if err != nil {
		re.SetError(err, cmdkit.ErrNormal)
		return
	}
	nodeId := n.Identity.Pretty()

	reportRequestItem := make(map[string]interface{})
	reportRequestItem["method"] = "addTemporaryPubKey"
	reportRequestItem["pubKey"] = base64.StdEncoding.EncodeToString(pubKeyBytes)
	reportRequestItem["nodeId"] = nodeId
	webServiceIp := "http://" + ip + ":" + commands.HithubPort + "/webservice/"
	responseResult, err := sendWebServiceRequest(reportRequestItem, webServiceIp, "POST")

	if err != nil {
		re.SetError(err, cmdkit.ErrNormal)
		return
	}
	responseValue, ok := responseResult["response"]
	if ok {
		if responseValue != "success" {
			fmt.Println("daemon start error!")
			return
		}
	} else {
		fmt.Println("There is something wrong with your request")
		return
	}
	// add by Nigel end

	fmt.Printf("Daemon is ready\n")

	// add by Nigel start: report the reposize
	count_reports := 0

	c := cron.New()
	spec := "0 */30 * * * ?" // every thirty minutes, and start from the 0 minute
	//spec := "*/5 * * * * ?"
	c.AddFunc(spec, func(){
		n, err := cmdenv.GetNode(env)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		sizeStat, err := corerepo.RepoSize(req.Context, n)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		// get node id
		nodeId := n.Identity.Pretty()

		_ = sizeStat
		_ = nodeId

		repoSize := sizeStat.RepoSize
		storageMax := sizeStat.StorageMax

		repoSizeString := strconv.FormatUint(repoSize, 10)
		storageMaxString := strconv.FormatUint(storageMax, 10)

		repoSizeBytes, err := sk.Sign([]byte(repoSizeString))
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		storageMaxBytes, err := sk.Sign([]byte(string(storageMaxString)))
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		reportRequestItem := make(map[string]interface{})
		reportRequestItem["method"] = "reportStorage"
		reportRequestItem["RepoSizeSign"] = base64.StdEncoding.EncodeToString(repoSizeBytes)
		reportRequestItem["StorageMaxSign"] = base64.StdEncoding.EncodeToString(storageMaxBytes)
		reportRequestItem["RepoSize"] = repoSizeString
		reportRequestItem["StorageMax"] = storageMaxString
		reportRequestItem["nodeId"] = nodeId
		bytesData, err := json.Marshal(reportRequestItem)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		reader := bytes.NewReader(bytesData)
		url := "http://47.105.76.115:8000/webservice/"
		request, err := http.NewRequest("POST", url, reader)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		request.Header.Set("Content-Type", "application/json;charset=UTF-8")
		client := http.Client{}
		resp, err := client.Do(request)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		defer resp.Body.Close()
		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		var mapResult map[string]interface{}
		if err := json.Unmarshal([]byte(string(respBytes)), &mapResult); err != nil {
			re.SetError(err, cmdkit.ErrNormal)
		}
		response, ok := mapResult["response"]
		if !ok {
			fmt.Println("something went wrong")
			return
		} else {
			if response != "success" {
				fmt.Println("something went wrong")
				return
			} else {
				count_reports += 1 // successfully get the response from the server
			}
		}
	})
	c.Start()
	select{}
	// add by Nigel end


	// collect long-running errors and block for shutdown
	// TODO(cryptix): our fuse currently doesnt follow this pattern for graceful shutdown
	for err := range merge(apiErrc, gwErrc, gcErrc) {
		if err != nil {
			log.Error(err)
			re.SetError(err, cmdkit.ErrNormal)
		}
	}
}

// serveHTTPApi collects options, creates listener, prints status message and starts serving requests
func serveHTTPApi(req *cmds.Request, cctx *oldcmds.Context) (<-chan error, error) {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: GetConfig() failed: %s", err)
	}

	apiAddr, _ := req.Options[commands.ApiOption].(string)
	if apiAddr == "" {
		apiAddr = cfg.Addresses.API
	}
	apiMaddr, err := ma.NewMultiaddr(apiAddr)
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: invalid API address: %q (err: %s)", apiAddr, err)
	}

	apiLis, err := manet.Listen(apiMaddr)
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: manet.Listen(%s) failed: %s", apiMaddr, err)
	}
	// we might have listened to /tcp/0 - lets see what we are listing on
	apiMaddr = apiLis.Multiaddr()
	fmt.Printf("API server listening on %s\n", apiMaddr)

	// by default, we don't let you load arbitrary ipfs objects through the api,
	// because this would open up the api to scripting vulnerabilities.
	// only the webui objects are allowed.
	// if you know what you're doing, go ahead and pass --unrestricted-api.
	unrestricted, _ := req.Options[unrestrictedApiAccessKwd].(bool)
	gatewayOpt := corehttp.GatewayOption(false, corehttp.WebUIPaths...)
	if unrestricted {
		gatewayOpt = corehttp.GatewayOption(true, "/ipfs", "/ipns")
	}

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("api"),
		corehttp.CheckVersionOption(),
		corehttp.CommandsOption(*cctx),
		corehttp.WebUIOption,
		gatewayOpt,
		corehttp.VersionOption(),
		defaultMux("/debug/vars"),
		defaultMux("/debug/pprof/"),
		corehttp.MetricsScrapingOption("/debug/metrics/prometheus"),
		corehttp.LogOption(),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: ConstructNode() failed: %s", err)
	}

	if err := node.Repo.SetAPIAddr(apiMaddr); err != nil {
		return nil, fmt.Errorf("serveHTTPApi: SetAPIAddr() failed: %s", err)
	}

	errc := make(chan error)
	go func() {
		errc <- corehttp.Serve(node, manet.NetListener(apiLis), opts...)
		close(errc)
	}()
	return errc, nil
}

// printSwarmAddrs prints the addresses of the host
func printSwarmAddrs(node *core.IpfsNode) {
	if !node.OnlineMode() {
		fmt.Println("Swarm not listening, running in offline mode.")
		return
	}

	var lisAddrs []string
	ifaceAddrs, err := node.PeerHost.Network().InterfaceListenAddresses()
	if err != nil {
		log.Errorf("failed to read listening addresses: %s", err)
	}
	for _, addr := range ifaceAddrs {
		lisAddrs = append(lisAddrs, addr.String())
	}
	sort.Sort(sort.StringSlice(lisAddrs))
	for _, addr := range lisAddrs {
		fmt.Printf("Swarm listening on %s\n", addr)
	}

	var addrs []string
	for _, addr := range node.PeerHost.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Sort(sort.StringSlice(addrs))
	for _, addr := range addrs {
		fmt.Printf("Swarm announcing %s\n", addr)
	}

}

// serveHTTPGateway collects options, creates listener, prints status message and starts serving requests
func serveHTTPGateway(req *cmds.Request, cctx *oldcmds.Context) (<-chan error, error) {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPGateway: GetConfig() failed: %s", err)
	}

	gatewayMaddr, err := ma.NewMultiaddr(cfg.Addresses.Gateway)
	if err != nil {
		return nil, fmt.Errorf("serveHTTPGateway: invalid gateway address: %q (err: %s)", cfg.Addresses.Gateway, err)
	}

	writable, writableOptionFound := req.Options[writableKwd].(bool)
	if !writableOptionFound {
		writable = cfg.Gateway.Writable
	}

	gwLis, err := manet.Listen(gatewayMaddr)
	if err != nil {
		return nil, fmt.Errorf("serveHTTPGateway: manet.Listen(%s) failed: %s", gatewayMaddr, err)
	}
	// we might have listened to /tcp/0 - lets see what we are listing on
	gatewayMaddr = gwLis.Multiaddr()

	if writable {
		fmt.Printf("Gateway (writable) server listening on %s\n", gatewayMaddr)
	} else {
		fmt.Printf("Gateway (readonly) server listening on %s\n", gatewayMaddr)
	}

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("gateway"),
		corehttp.CheckVersionOption(),
		corehttp.CommandsROOption(*cctx),
		corehttp.VersionOption(),
		corehttp.IPNSHostnameOption(),
		corehttp.GatewayOption(writable, "/ipfs", "/ipns"),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPGateway: ConstructNode() failed: %s", err)
	}

	errc := make(chan error)
	go func() {
		errc <- corehttp.Serve(node, manet.NetListener(gwLis), opts...)
		close(errc)
	}()
	return errc, nil
}

//collects options and opens the fuse mountpoint
func mountFuse(req *cmds.Request, cctx *oldcmds.Context) error {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return fmt.Errorf("mountFuse: GetConfig() failed: %s", err)
	}

	fsdir, found := req.Options[ipfsMountKwd].(string)
	if !found {
		fsdir = cfg.Mounts.IPFS
	}

	nsdir, found := req.Options[ipnsMountKwd].(string)
	if !found {
		nsdir = cfg.Mounts.IPNS
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return fmt.Errorf("mountFuse: ConstructNode() failed: %s", err)
	}

	err = nodeMount.Mount(node, fsdir, nsdir)
	if err != nil {
		return err
	}
	fmt.Printf("IPFS mounted at: %s\n", fsdir)
	fmt.Printf("IPNS mounted at: %s\n", nsdir)
	return nil
}

func maybeRunGC(req *cmds.Request, node *core.IpfsNode) (<-chan error, error) {
	enableGC, _ := req.Options[enableGCKwd].(bool)
	if !enableGC {
		return nil, nil
	}

	errc := make(chan error)
	go func() {
		errc <- corerepo.PeriodicGC(req.Context, node)
		close(errc)
	}()
	return errc, nil
}

// merge does fan-in of multiple read-only error channels
// taken from http://blog.golang.org/pipelines
func merge(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	for _, c := range cs {
		if c != nil {
			wg.Add(1)
			go output(c)
		}
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func YesNoPrompt(prompt string) bool {
	var s string
	for i := 0; i < 3; i++ {
		fmt.Printf("%s ", prompt)
		fmt.Scanf("%s", &s)
		switch s {
		case "y", "Y":
			return true
		case "n", "N":
			return false
		case "":
			return false
		}
		fmt.Println("Please press either 'y' or 'n'")
	}

	return false
}
