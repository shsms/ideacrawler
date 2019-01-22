/*************************************************************************
 *
 * Copyright 2018 Ideas2IT Technology Services Private Limited.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 ***********************************************************************/

package main

import (
	"errors"
	"log"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"path"
	"strings"
	"time"

	"github.com/shsms/sflag"
)

// TODO - implement the DEBUG feature
var cliParams = struct {
	DialAddress          string "Interface to dial from.  Defaults to OS defined routes|"
	SaveLoginPages       string "Saves login pages in this path. Don't save them if empty."
	ClientListenAddress  string "Interface to listen on. Defaults to 127.0.0.1|127.0.0.1:2345"
	ClusterListenAddress string "Interface to listen on. Defaults to 127.0.0.1|127.0.0.1:2346"
	LogPath              string "Logpath to log into. Default is stdout."
	Mode                 string "Instance mode - standalone/worker/server|standalone"
	Servers              string "In worker mode, the hostname of server to connect to."
}{}

const (
	modeStandalone = iota
	modeWorker
	modeServer
)

var (
	errInvalidMode = errors.New("invalid mode.")
)

func parseCrawlerMode(mode string) (int, error) {
	modes := map[string]int{
		"standalone": modeStandalone,
		"worker":     modeWorker,
		"server":     modeServer,
	}
	modeClean := strings.ToLower(strings.TrimSpace(mode))
	if m, ok := modes[modeClean]; ok {
		return m, nil
	}
	return -1, errInvalidMode
}

type cancelSignal struct{}
type jobDoneSignal struct{}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	rand.Seed(time.Now().UTC().UnixNano())

	sflag.Parse(&cliParams)
	if cliParams.LogPath != "" {
		err := os.MkdirAll(cliParams.LogPath, 0755)
		if err != nil {
			panic(err)
		}

		logFP, err := os.Create(path.Join(cliParams.LogPath, "master.log"))
		if err != nil {
			panic(err)
		}
		log.SetOutput(logFP)
	}
	mode, err := parseCrawlerMode(cliParams.Mode)
	if err == errInvalidMode {
		log.Fatal(err)
	}
	if mode == modeServer {
		joinServerCluster()
	} else {
		startCrawlerWorker(mode)
	}
}
