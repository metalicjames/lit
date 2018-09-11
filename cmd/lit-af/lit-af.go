package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

/*
Lit-AF

The Lit Advanced Functionality interface.
This is a text mode interface to lit.  It connects over jsonrpc to the a lit
node and tells that lit node what to do.  The lit node also responds so that
lit-af can tell what's going on.

lit-gtk does most of the same things with a gtk interface, but there will be
some yet-undefined advanced functionality only available in lit-af.

May end up using termbox-go

*/

//// BalReply is the reply when the user asks about their balance.
//// This is a Non-Channel
//type BalReply struct {
//	ChanTotal         int64
//	TxoTotal          int64
//	SpendableNow      int64
//	SpendableNowWitty int64
//}

const (
	litHomeDirName     = ".lit"
	historyFilename    = "lit-af.history"
	defaultKeyFileName = "privkey.hex"
)

type litAfClient struct {
	con           string
	tracker       string
	litHomeDir    string
	lndcRpcClient *litrpc.LndcRpcClient
}

type Command struct {
	Format           string
	Description      string
	ShortDescription string
}

func setConfig(lc *litAfClient) {
	conptr := flag.String("con", "2448", "host to connect to in the form of [<lnadr>@][<host>][:<port>]")
	dirptr := flag.String("dir", filepath.Join(os.Getenv("HOME"), litHomeDirName), "directory to save settings")
	trackerptr := flag.String("tracker", "http://hubris.media.mit.edu:46580", "service to use for looking up node addresses")

	flag.Parse()

	lc.con = *conptr
	lc.tracker = *trackerptr
	lc.litHomeDir = *dirptr
}

// for now just testing how to connect and get messages back and forth
func main() {
	var err error

	lc := new(litAfClient)
	setConfig(lc)

	// create home directory if it does not exist
	_, err = os.Stat(lc.litHomeDir)
	if os.IsNotExist(err) {
		os.Mkdir(lc.litHomeDir, 0700)
	}

	adr, host, port := lnutil.ParseAdrStringWithPort(lc.con)

	if litrpc.LndcRpcCanConnectLocallyWithHomeDir(lc.litHomeDir) && adr == "" && (host == "localhost" || host == "127.0.0.1") {

		lc.lndcRpcClient, err = litrpc.NewLocalLndcRpcClientWithHomeDirAndPort(lc.litHomeDir, port)
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		if !lnutil.LitAdrOK(adr) {
			log.Fatal("lit address passed in -con parameter is not valid")
		}

		keyFilePath := filepath.Join(lc.litHomeDir, "lit-af-key.hex")
		privKey, err := lnutil.ReadKeyFile(keyFilePath)
		if err != nil {
			log.Fatal(err.Error())
		}
		key, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey[:])

		if adr != "" && strings.HasPrefix(adr, "ln1") && host == "" {
			ipv4, _, err := lnutil.Lookup(adr, lc.tracker, "")
			if err != nil {
				log.Fatalf("Error looking up address on the tracker: %s", err)
			} else {
				adr = fmt.Sprintf("%s@%s", adr, ipv4)
			}
		} else {
			adr = fmt.Sprintf("%s@%s:%d", adr, host, port)
		}

		log.Printf("Connecting to %s\n", adr)

		lc.lndcRpcClient, err = litrpc.NewLndcRpcClient(adr, key)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       lnutil.Prompt("lit-af") + lnutil.White("# "),
		HistoryFile:  filepath.Join(lc.litHomeDir, historyFilename),
		AutoComplete: lc.NewAutoCompleter(),
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	// main shell loop
	for {
		// setup reader with max 4K input chars
		msg, err := rl.Readline()
		if err != nil {
			break
		}
		msg = strings.TrimSpace(msg)
		if len(msg) == 0 {
			continue
		}
		rl.SaveHistory(msg)

		cmdslice := strings.Fields(msg)                         // chop input up on whitespace
		fmt.Fprintf(color.Output, "entered command: %s\n", msg) // immediate feedback

		err = lc.Shellparse(cmdslice)
		if err != nil { // only error should be user exit
			panic(err)
		}
	}

	//	err = c.Call("LitRPC.Bal", nil, &br)
	//	if err != nil {
	//		Log.Fatal("rpc call error:", err)
	//	}
	//	fmt.Printf("Sent bal req, response: txototal %d\n", br.TxoTotal)
}

func (lc *litAfClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return lc.lndcRpcClient.Call(serviceMethod, args, reply)
}
