package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ipfs/go-ipfs/cmd/ipfs/util"
	"github.com/ipfs/go-ipfs/core/commands"
	"github.com/sparrc/go-ping"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ipfs/go-ipfs/assets"
	oldcmds "github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/repo/fsrepo"

	"gx/ipfs/QmPTfgFTo9PFr1PvPKyKoeMgBvYPh6cX3aDP7DHKVbnCbi/go-ipfs-cmds"
	"gx/ipfs/QmSP88ryZkHSRn1fnngAaV2Vcn63WUJzAavnRM9CVdU1Ky/go-ipfs-cmdkit"
	"gx/ipfs/QmTyiSs9VgdVb4pnzdjtKhcfdTkHFEaNn6xnCbZq4DTFRt/go-ipfs-config"
)

var initCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Initializes ipfs config file.",
		ShortDescription: `
Initializes ipfs configuration files and generates a new keypair.

If you are going to run IPFS in server environment, you may want to
initialize it using 'server' profile.

For the list of available profiles see 'ipfs config profile --help'

ipfs uses a repository in the local file system. By default, the repo is
located at ~/.ipfs. To change the repo location, set the $IPFS_PATH
environment variable:

    export IPFS_PATH=/path/to/ipfsrepo
`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.FileArg("default-config", false, false, "Initialize with the given configuration.").EnableStdin(),
	},
	Options: []cmdkit.Option{
		cmdkit.IntOption("bits", "b", "Number of bits to use in the generated RSA private key.").WithDefault(commands.NBitsForKeypairDefault),
		cmdkit.BoolOption("empty-repo", "e", "Don't add and pin help files to the local storage."),
		cmdkit.StringOption("profile", "p", "Apply profile settings to config. Multiple profiles can be separated by ','"),

		// TODO need to decide whether to expose the override as a file or a
		// directory. That is: should we allow the user to also specify the
		// name of the file?
		// TODO cmdkit.StringOption("event-logs", "l", "Location for machine-readable event logs."),
	},
	PreRun: func(req *cmds.Request, env cmds.Environment) error {
		cctx := env.(*oldcmds.Context)
		daemonLocked, err := fsrepo.LockedByOtherProcess(cctx.ConfigRoot)
		if err != nil {
			return err
		}

		log.Info("checking if daemon is running...")
		if daemonLocked {
			log.Debug("ipfs daemon is running")
			e := "ipfs daemon is running. please stop it to run this command"
			return cmds.ClientError(e)
		}

		return nil
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) {
		cctx := env.(*oldcmds.Context)
		if cctx.Online {
			res.SetError(errors.New("init must be run offline only"), cmdkit.ErrNormal)
			return
		}

		empty, _ := req.Options["empty-repo"].(bool)
		nBitsForKeypair, _ := req.Options["bits"].(int)

		var conf *config.Config

		f := req.Files
		if f != nil {
			confFile, err := f.NextFile()
			if err != nil {
				res.SetError(err, cmdkit.ErrNormal)
				return
			}

			conf = &config.Config{}
			if err := json.NewDecoder(confFile).Decode(conf); err != nil {
				res.SetError(err, cmdkit.ErrNormal)
				return
			}
		}

		profile, _ := req.Options["profile"].(string)

		var profiles []string
		if profile != "" {
			profiles = strings.Split(profile, ",")
		}

		//// add by Nigel start: get the arguments
		//var params = req.Options
		//var serverIp string
		//if value, ok := params["serverIp"]; ok {
		//	serverIp = value.(string)
		//	_ = serverIp
		//}
		//var serverPort string
		//if value, ok := params["serverPort"]; ok {
		//	serverPort = value.(string)
		//	_ = serverPort
		//}
		//if serverIp == "" {
		//	fmt.Println("no server ip")
		//	return
		//}
		//if serverPort == "" {
		//	fmt.Println("no server port")
		//	return
		//}
		//
		//// add by Nigel end

		if err := doInit(os.Stdout, cctx.ConfigRoot, empty, nBitsForKeypair, profiles, conf); err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}
	},
}

var errRepoExists = errors.New(`ipfs configuration file already exists!
Reinitializing would overwrite your keys.
`)

func initWithDefaults(out io.Writer, repoRoot string, profile string) error {
	var profiles []string
	if profile != "" {
		profiles = strings.Split(profile, ",")
	}

	return doInit(out, repoRoot, false, commands.NBitsForKeypairDefault, profiles, nil)
}

// add by Nigel start: get ping milliseconds
func getPingMilliseconds(ip string) float64{
	delay := -1.0
	pinger, err := ping.NewPinger(ip)
	pinger.Timeout = time.Second * 5 // timeout in 5 seconds
	if err != nil {
		return delay
	}
	pinger.OnRecv = func(pkt *ping.Packet) {
		delay = pkt.Rtt.Seconds() * 1000 // milliseconds
		pinger.Stop()
	}
	pinger.Run()
	return delay
}
// add by Nigel end

// add by Nigel start:
func New(path string, mode os.FileMode) (*commands.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(f.Name(), mode); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	return &commands.File{File: f, Path: path}, nil
}
// add by Nigel end

// add by Nigel start: encode configuration with json
func encode(w io.Writer, value interface{}) error {
	buf, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}
// add by Nigel end

// add by Nigel start: write config file
func WriteConfigFile(filename string, cfg interface{}) error {
	err := os.MkdirAll(filepath.Dir(filename), 0775)
	if err != nil {
		return err
	}

	f, err := New(filename, 0660)
	if err != nil {
		return err
	}
	defer f.Close()

	return encode(f, cfg)
}
// add by Nigel end

func doInit(out io.Writer, repoRoot string, empty bool, nBitsForKeypair int, confProfiles []string, conf *config.Config) error {

	if _, err := fmt.Fprintf(out, "initializing IPFS node at %s\n", repoRoot); err != nil {
		return err
	}

	if err := checkWritable(repoRoot); err != nil {
		return err
	}

	if fsrepo.IsInitialized(repoRoot) {
		return errRepoExists
	}


	// add by Nigel start: getAllServers that can be connected
	var selectIp string
	webServiceIp := "http://" + commands.HithubIp + ":" + commands.HithubPort + "/webservice/"
	reportRequestItem := make(map[string]interface{})
	reportRequestItem["method"] = "getAllServers"
	responseResult, err := sendWebServiceRequest(reportRequestItem, webServiceIp, "POST")
	if err != nil {
		fmt.Println("Error with the network!")
		return nil
	}
	responseValue, ok := responseResult["response"]
	if ok {
		if responseValue != "success" {
			return nil
		} else {
			ipListStr, ok := responseResult["ipList"]
			if !ok {
				return nil
			}
			addressListStr, ok := responseResult["addressList"]
			if !ok {
				return nil
			}
			ipArray := strings.Split(ipListStr.(string), ".,.")
			addressArray := strings.Split(addressListStr.(string), ".,.")
			num := len(ipArray)
			if num > 0 {
				fmt.Println("Enter the number of server from the list below:")
			} else {
				fmt.Println("No server exists, cannot init a client!")
				return nil
			}
			var serverArray = make([]string, num)
			for i := 0; i < num; i++ {
				ip := ipArray[i]
				address := addressArray[i]
				delay := getPingMilliseconds(ip)
				if delay < 0 {
					continue
				}
				fmt.Printf("%d: %s, delay: %.2fms\n", i + 1, address, delay)
				serverArray[i] = ip
			}
			var whichServer string
			fmt.Scanln(&whichServer)
			whichServerInt, err:=strconv.Atoi(whichServer)
			if err != nil || whichServerInt <= 0 || whichServerInt > num {
				fmt.Println("Error with the input number!")
				return nil
			}
			selectIp = serverArray[whichServerInt - 1]
		}
	} else {
		fmt.Println("There is something wrong with your request")
		return nil
	}
	// add by Nigel end

	// add by Nigel start: verify username and password
	var username, password string
	fmt.Println("Please insert the username and password of Hithub (ctrl+c to exit):")
	fmt.Print("username: ")
	fmt.Scanln(&username)
	fmt.Print("password: ")
	fmt.Scanln(&password)
	// judge whether the username and password correct
	reportRequestItem = make(map[string]interface{})
	reportRequestItem["method"] = "checkUserPassword"
	reportRequestItem["username"] = username
	reportRequestItem["password"] = password
	webServiceIp = "http://" + commands.HithubIp + ":" + commands.HithubPort + "/webservice/"
	responseResult, err = sendWebServiceRequest(reportRequestItem, webServiceIp, "POST")
	if err != nil {
		fmt.Println("Error with the network!")
		return nil
	}
	responseValue, ok = responseResult["response"]
	if ok {
		if responseValue != "success" {
			fmt.Println("Username and password do not match!")
			return nil
		}
	} else {
		fmt.Println("There is something wrong with your request")
		return nil
	}
	// add by Nigel end

	if conf == nil {
		var err error
		conf, err = config.Init(out, nBitsForKeypair)
		if err != nil {
			return err
		}
		// add by Nigel start: init with username
		reportRequestItem := make(map[string]interface{})
		reportRequestItem["method"] = "initWithUsername"
		reportRequestItem["username"] = username
		reportRequestItem["password"] = password
		reportRequestItem["nodeId"] = conf.Identity.PeerID
		responseResult, err := sendWebServiceRequest(reportRequestItem, webServiceIp, "POST")
		if err != nil {
			fmt.Println("Error with the network!")
			return nil
		}
		responseValue, ok := responseResult["response"]
		if ok {
			if responseValue != "success" {
				fmt.Println("Username and password do not match!")
				return nil
			}
		} else {
			fmt.Println("There is something wrong with your request")
			return nil
		}
		// add by Nigel end

		// add by Nigel start: register client to server's listener
		err = util.HitListenerAdd(selectIp, commands.ServerListenerPort, conf.Identity.PeerID)
		if err != nil {
			return nil
		}
		// add by Nigel end

	}

	for _, profile := range confProfiles {
		transformer, ok := config.Profiles[profile]
		if !ok {
			return fmt.Errorf("invalid configuration profile: %s", profile)
		}

		if err := transformer.Transform(conf); err != nil {
			return err
		}
	}

	if err := fsrepo.Init(repoRoot, conf); err != nil {
		return err
	}

	if !empty {
		if err := addDefaultAssets(out, repoRoot); err != nil {
			return err
		}
	}

	// add by Nigel start: write a file
	//var serverFile *os.File
	//var err1 error
	var hitconfig commands.HitConfig
	hitconfig.Init(selectIp, "1.0")
	WriteConfigFile(path.Join(repoRoot, commands.ClientFileName), hitconfig)
	return nil
	//if commands.CheckFileIsExist(path.Join(repoRoot, commands.ClientFileName)) {
	//	err := os.Remove(path.Join(repoRoot, commands.ClientFileName))
	//	if err != nil{
	//		return err
	//	}
	//	serverFile, err1 = os.OpenFile(path.Join(repoRoot, commands.ClientFileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666) //打开文件
	//	if err1 != nil{
	//		return err1
	//	}
	//} else {
	//	serverFile, err1 = os.Create(path.Join(repoRoot, commands.ClientFileName))
	//	if err1 != nil{
	//		return err1
	//	}
	//}
	//clientFileContent := selectIp
	//n, err1 := io.WriteString(serverFile, clientFileContent)
	//if err1 != nil{
	//	return err1
	//}
	//_ = n
	// add by Nigel end

	// add by Nigel start: add swarm.key file into .ipfs directory
	if commands.CheckFileIsExist(path.Join(repoRoot, "swarm.key")) {
		return errors.New("file already exists")
	} else {
		swarmKeyFile, err1 := os.Create(path.Join(repoRoot, "swarm.key"))
		if err1 != nil{
			return err1
		}
		swarmKeyFileContent := commands.SwarmKeyContent
		n, err1 := io.WriteString(swarmKeyFile, swarmKeyFileContent)
		if err1 != nil{
			return err1
		}
		_ = n
	}
	// add by Nigel end

	return initializeIpnsKeyspace(repoRoot)
}

func checkWritable(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		// dir exists, make sure we can write to it
		testfile := path.Join(dir, "test")
		fi, err := os.Create(testfile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%s is not writeable by the current user", dir)
			}
			return fmt.Errorf("unexpected error while checking writeablility of repo root: %s", err)
		}
		fi.Close()
		return os.Remove(testfile)
	}

	if os.IsNotExist(err) {
		// dir doesn't exist, check that we can create it
		return os.Mkdir(dir, 0775)
	}

	if os.IsPermission(err) {
		return fmt.Errorf("cannot write to %s, incorrect permissions", err)
	}

	return err
}

func addDefaultAssets(out io.Writer, repoRoot string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := fsrepo.Open(repoRoot)
	if err != nil { // NB: repo is owned by the node
		return err
	}

	nd, err := core.NewNode(ctx, &core.BuildCfg{Repo: r})
	if err != nil {
		return err
	}
	defer nd.Close()

	dkey, err := assets.SeedInitDocs(nd)
	if err != nil {
		return fmt.Errorf("init: seeding init docs failed: %s", err)
	}
	log.Debugf("init: seeded init docs %s", dkey)

	if _, err = fmt.Fprintf(out, "to get started, enter:\n"); err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, "\n\tipfs cat /ipfs/%s/readme\n\n", dkey)
	return err
}

func initializeIpnsKeyspace(repoRoot string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := fsrepo.Open(repoRoot)
	if err != nil { // NB: repo is owned by the node
		return err
	}

	nd, err := core.NewNode(ctx, &core.BuildCfg{Repo: r})
	if err != nil {
		return err
	}
	defer nd.Close()

	err = nd.SetupOfflineRouting()
	if err != nil {
		return err
	}

	return namesys.InitializeKeyspace(ctx, nd.Namesys, nd.Pinning, nd.PrivateKey)
}
