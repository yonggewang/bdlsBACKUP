// BSD 3-Clause License
//
// Copyright (c) 2020, Sperax
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its
//    contributors may be used to endorse or promote products derived from
//    this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package agent

import (
	"crypto/ecdsa"
	"net"
	"sync"
	"time"
)

const (
	defaultWriteTimeout = 10 * time.Second
)

// TCPPeer contains information related to a tcp connection
type TCPPeer struct {
	conn      net.Conn         // the connection to this peer
	publicKey *ecdsa.PublicKey // if it's not nil, the peer is known(authenticated in some way)

	pending   [][]byte      // pending output data to this peer
	chPending chan struct{} // notification on new pending data

	die        chan struct{}
	sync.Mutex // mutex for all fields
}

// NewTCPPeer creates a consensus peer based on net.Conn and and async-io(gaio) watcher for sending
func NewTCPPeer(conn net.Conn) *TCPPeer {
	p := new(TCPPeer)
	p.chPending = make(chan struct{}, 1)
	p.conn = conn
	p.die = make(chan struct{})
	return p
}

// GetPublicKey returns peer's public key as identity
func (p *TCPPeer) GetPublicKey() *ecdsa.PublicKey {
	p.Lock()
	defer p.Unlock()
	return p.publicKey
}

// RemoteAddr should return peer's address as identity
func (p *TCPPeer) RemoteAddr() net.Addr { return p.conn.RemoteAddr() }

// Send message to this peer
func (p *TCPPeer) Send(out []byte) error {
	p.Lock()
	defer p.Unlock()
	p.pending = append(p.pending, out)
	p.notifyPending()
	return nil
}

// notifyPending output
func (p *TCPPeer) notifyPending() {
	select {
	case p.chPending <- struct{}{}:
	default:
	}
}

// a loop for transmission
func (p *TCPPeer) txLoop() {
	for {
		select {
		case <-p.chPending:
		case <-p.die:
			return
		}
	}
}