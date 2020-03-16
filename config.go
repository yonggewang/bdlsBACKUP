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

package consensus

import (
	"crypto/ecdsa"
	"time"
)

const (
	// ConfigMinimumParticipants is the minimum number of participant allow in consensus protocol
	ConfigMinimumParticipants = 4
)

// Config is to config the parameters of BDLS consensus protocol
type Config struct {
	// the starting time point for consensus
	Epoch time.Time
	// CurrentHeight
	CurrentHeight uint64
	// CurrentState
	CurrentState State
	// PrivateKey
	PrivateKey *ecdsa.PrivateKey
	// Consensus Group
	Participants []*ecdsa.PublicKey

	// StateCompare is a function from user to compare states,
	// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
	// Ususally this would be block header in blockchain, or replication log in database,
	// users should check fields in block header to make comparsion.
	StateCompare func(a State, b State) int

	// StateValidate is a function from user to validate the integrity of
	// a state data.
	StateValidate func(State) bool

	// StateHash is a function from user to return a hash to uniquely identifies
	// a state.
	StateHash func(State) StateHash
}

// VerifyConfig verifies the integrity of this config when creating new consensus object
func VerifyConfig(c *Config) error {
	if c.Epoch.IsZero() {
		return ErrConfigEpoch
	}

	if c.CurrentState == nil {
		return ErrConfigStateNil
	}

	if c.StateCompare == nil {
		return ErrConfigLess
	}

	if c.StateValidate == nil {
		return ErrConfigValidateState
	}

	if c.PrivateKey == nil {
		return ErrConfigPrivateKey
	}

	if len(c.Participants) < ConfigMinimumParticipants {
		return ErrConfigParticipants
	}

	return nil
}