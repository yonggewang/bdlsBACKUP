// Copyright (c) 2020 Sperax
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/xtaci/bdls/agent-tcp"
	"github.com/xtaci/bdls/consensus"
)

// A quorum set for consenus
type Quorum struct {
	State  []byte   `json:"state"`
	Height uint64   `json:"height"`
	Keys   [][]byte `json:"keys"` // pem formatted keys
}

func main() {
	app := &cli.App{
		Name:                 "BDLS consensus protocol test client",
		Usage:                "Act as a participant to BDLS consensus quorum",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:  "genkeys",
				Usage: "generate keys to participant",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "count",
						Value: 5,
						Usage: "number of participant to generate",
					},
					&cli.StringFlag{
						Name:  "config",
						Value: "./quorum.json",
						Usage: "output quorum file, all participants will use this",
					},
				},
				Action: func(c *cli.Context) error {
					count := c.Int("count")
					quorum := &Quorum{}
					// generate private keys
					for i := 0; i < count; i++ {
						privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
						if err != nil {
							return err
						}
						bts, err := x509.MarshalECPrivateKey(privateKey)
						if err != nil {
							return err
						}
						quorum.Keys = append(quorum.Keys, bts)
					}

					// generate a random state
					initialData := make([]byte, 1024)
					_, err := io.ReadFull(rand.Reader, initialData)
					if err != nil {
						return err
					}
					quorum.State = initialData
					quorum.Height = 0

					file, err := os.Create(c.String("config"))
					if err != nil {
						return err
					}
					enc := json.NewEncoder(file)
					enc.SetIndent("", "\t")
					err = enc.Encode(quorum)
					if err != nil {
						return err
					}
					file.Close()

					log.Println("generate", c.Int("count"), "keys")
					return nil
				},
			},
			{
				Name:  "run",
				Usage: "start a consensus agent",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "listen",
						Value: ":4680",
						Usage: "the client's listening port",
					},
					&cli.IntFlag{
						Name:  "id",
						Value: 0,
						Usage: "the node id, will use the n-th private key in quorum.json",
					},
					&cli.StringFlag{
						Name:  "config",
						Value: "./quorum.json",
						Usage: "the shared quorum config file",
					},
					&cli.StringFlag{
						Name:  "peers",
						Value: "./peers.json",
						Usage: "the peers ip list to connect",
					},
				},
				Action: func(c *cli.Context) error {
					file, err := os.Open(c.String("config"))
					if err != nil {
						return err
					}
					defer file.Close()

					quorum := new(Quorum)
					err = json.NewDecoder(file).Decode(quorum)
					if err != nil {
						return err
					}

					id := c.Int("id")
					if id >= len(quorum.Keys) {
						return errors.New(fmt.Sprint("cannot locate private key for id:", id))
					}

					// open peers
					file, err = os.Open(c.String("peers"))
					if err != nil {
						return err
					}
					defer file.Close()

					var peers []string
					err = json.NewDecoder(file).Decode(&peers)
					if err != nil {
						return err
					}

					for k := range peers {
						log.Println("peer", k, peers[k])
					}

					// use config for this id and set less function
					config := new(consensus.Config)
					config.Epoch = time.Now()
					config.CurrentState = quorum.State
					config.CurrentHeight = quorum.Height

					for k := range quorum.Keys {
						priv, err := x509.ParseECPrivateKey(quorum.Keys[k])
						if err != nil {
							return err
						}

						// myself
						if id == k {
							config.PrivateKey = priv
						}

						config.Participants = append(config.Participants, &priv.PublicKey)
					}

					config.StateCompare = func(a consensus.State, b consensus.State) int { return bytes.Compare(a, b) }

					config.StateValidate = func(consensus.State) bool { return true }

					tcpaddr, err := net.ResolveTCPAddr("tcp", c.String("listen"))
					if err != nil {
						return err
					}

					l, err := net.ListenTCP("tcp", tcpaddr)
					if err != nil {
						return err
					}
					log.Println("listening on:", tcpaddr)

					log.Println("consenus started")
					agent, err := agent.NewAgent(l, config)
					if err != nil {
						return err
					}

					// background connect peers
					for k := range peers {
						go func(raddr string) {
							for {
								conn, err := net.Dial("tcp", raddr)
								if err == nil {
									agent.AddPeer(conn.(*net.TCPConn))
									return
								}
								log.Println(err)
								<-time.After(time.Second)
							}
						}(peers[k])
					}

					for {
						data := make([]byte, 1024)
						io.ReadFull(rand.Reader, data)
						agent.Propose(data)

						// wait until next height
						confirmedStates, err := agent.Wait()
						if err != nil {
							return err
						}

						for _, cs := range confirmedStates {
							h := consensus.DefaultHash(cs.State)
							log.Printf("<decide> at height: %v, round: %v, hash:%v", cs.Height, cs.Round, hex.EncodeToString(h[:]))
						}
					}
				},
			},
		},

		Action: func(c *cli.Context) error {
			cli.ShowAppHelp(c)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}