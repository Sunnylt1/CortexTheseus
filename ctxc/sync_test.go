// Copyright 2019 The CortexTheseus Authors
// This file is part of the CortexFoundation library.
//
// The CortexFoundation library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The CortexFoundation library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the CortexFoundation library. If not, see <http://www.gnu.org/licenses/>.

package ctxc

import (
	"testing"
	"time"

	"github.com/CortexFoundation/CortexTheseus/ctxc/downloader"
	"github.com/CortexFoundation/CortexTheseus/p2p"
	"github.com/CortexFoundation/CortexTheseus/p2p/enode"
)

//func TestFastSyncDisabling63(t *testing.T) { testFastSyncDisabling(t, 63) }
//func TestFastSyncDisabling64(t *testing.T) { testFastSyncDisabling(t, 64) }
//func TestFastSyncDisabling65(t *testing.T) { testFastSyncDisabling(t, 65) }

// Tests that fast sync gets disabled as soon as a real block is successfully
// imported into the blockchain.
func testFastSyncDisabling(t *testing.T, protocol int) {
	t.Parallel()

	// Create a pristine protocol manager, check that fast sync is left enabled
	pmEmpty, _ := newTestProtocolManagerMust(t, downloader.FastSync, 0, nil, nil)
	if !pmEmpty.fastSync.Load() {
		t.Fatalf("fast sync disabled on pristine blockchain")
	}
	// Create a full protocol manager, check that fast sync gets disabled
	pmFull, _ := newTestProtocolManagerMust(t, downloader.FastSync, 1024, nil, nil)
	if pmFull.fastSync.Load() {
		t.Fatalf("fast sync not disabled on non-empty blockchain")
	}

	// Sync up the two peers
	io1, io2 := p2p.MsgPipe()
	go pmFull.handle(pmFull.newPeer(uint(protocol), p2p.NewPeer(enode.ID{}, "empty", nil), io2, pmFull.txpool.Get))
	go pmEmpty.handle(pmEmpty.newPeer(uint(protocol), p2p.NewPeer(enode.ID{}, "full", nil), io1, pmEmpty.txpool.Get))

	time.Sleep(250 * time.Millisecond)
	op := peerToSyncOp(downloader.FastSync, pmEmpty.peers.BestPeer())
	if err := pmEmpty.doSync(op); err != nil {
		t.Fatal("sync failed:", err)
	}

	// Check that fast sync was disabled
	if pmEmpty.fastSync.Load() {
		t.Fatalf("fast sync not disabled after successful synchronisation")
	}
}
