// Copyright 2019 The CortexTheseus Authors
// This file is part of CortexTheseus.
//
// CortexTheseus is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// CortexTheseus is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with CortexTheseus. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"net"
	"time"

	"github.com/CortexFoundation/CortexTheseus/core"
	"github.com/CortexFoundation/CortexTheseus/core/forkid"
	"github.com/CortexFoundation/CortexTheseus/p2p/enr"
	"github.com/CortexFoundation/CortexTheseus/params"
	"github.com/CortexFoundation/CortexTheseus/rlp"
	"gopkg.in/urfave/cli.v1"
)

var (
	nodesetCommand = cli.Command{
		Name:  "nodeset",
		Usage: "Node set tools",
		Subcommands: []cli.Command{
			nodesetInfoCommand,
			nodesetFilterCommand,
		},
	}
	nodesetInfoCommand = cli.Command{
		Name:      "info",
		Usage:     "Shows statistics about a node set",
		Action:    nodesetInfo,
		ArgsUsage: "<nodes.json>",
	}
	nodesetFilterCommand = cli.Command{
		Name:      "filter",
		Usage:     "Filters a node set",
		Action:    nodesetFilter,
		ArgsUsage: "<nodes.json> filters..",

		SkipFlagParsing: true,
	}
)

func nodesetInfo(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("need nodes file as argument")
	}

	ns := loadNodesJSON(ctx.Args().First())
	fmt.Printf("Set contains %d nodes.\n", len(ns))
	return nil
}

func nodesetFilter(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("need nodes file as argument")
	}
	ns := loadNodesJSON(ctx.Args().First())
	filter, err := andFilter(ctx.Args().Tail())
	if err != nil {
		return err
	}

	result := make(nodeSet)
	for id, n := range ns {
		if filter(n) {
			result[id] = n
		}
	}
	writeNodesJSON("-", result)
	return nil
}

type nodeFilter func(nodeJSON) bool

type nodeFilterC struct {
	narg int
	fn   func([]string) (nodeFilter, error)
}

var filterFlags = map[string]nodeFilterC{
	"-ip":          {1, ipFilter},
	"-min-age":     {1, minAgeFilter},
	"-eth-network": {1, ethFilter},
	"-les-server":  {0, lesFilter},
	"-snap":        {0, snapFilter},
}

func parseFilters(args []string) ([]nodeFilter, error) {
	var filters []nodeFilter
	for len(args) > 0 {
		fc, ok := filterFlags[args[0]]
		if !ok {
			return nil, fmt.Errorf("invalid filter %q", args[0])
		}
		if len(args)-1 < fc.narg {
			return nil, fmt.Errorf("filter %q wants %d arguments, have %d", args[0], fc.narg, len(args)-1)
		}
		filter, err := fc.fn(args[1 : 1+fc.narg])
		if err != nil {
			return nil, fmt.Errorf("%s: %v", args[0], err)
		}
		filters = append(filters, filter)
		args = args[1+fc.narg:]
	}
	return filters, nil
}

func andFilter(args []string) (nodeFilter, error) {
	checks, err := parseFilters(args)
	if err != nil {
		return nil, err
	}
	f := func(n nodeJSON) bool {
		for _, filter := range checks {
			if !filter(n) {
				return false
			}
		}
		return true
	}
	return f, nil
}

func ipFilter(args []string) (nodeFilter, error) {
	_, cidr, err := net.ParseCIDR(args[0])
	if err != nil {
		return nil, err
	}
	f := func(n nodeJSON) bool { return cidr.Contains(n.N.IP()) }
	return f, nil
}

func minAgeFilter(args []string) (nodeFilter, error) {
	minage, err := time.ParseDuration(args[0])
	if err != nil {
		return nil, err
	}
	f := func(n nodeJSON) bool {
		age := n.LastResponse.Sub(n.FirstResponse)
		return age >= minage
	}
	return f, nil
}

func ethFilter(args []string) (nodeFilter, error) {
	var filter forkid.Filter
	switch args[0] {
	case "mainnet":
		filter = forkid.NewStaticFilter(params.MainnetChainConfig, core.DefaultGenesisBlock().ToBlock(nil))
	default:
		return nil, fmt.Errorf("unknown network %q", args[0])
	}

	f := func(n nodeJSON) bool {
		var eth struct {
			ForkID forkid.ID
			Tail   []rlp.RawValue `rlp:"tail"`
		}
		if n.N.Load(enr.WithEntry("eth", &eth)) != nil {
			return false
		}
		return filter(eth.ForkID) == nil
	}
	return f, nil
}

func lesFilter(args []string) (nodeFilter, error) {
	f := func(n nodeJSON) bool {
		var les struct {
			Tail []rlp.RawValue `rlp:"tail"`
		}
		return n.N.Load(enr.WithEntry("les", &les)) == nil
	}
	return f, nil
}

func snapFilter(args []string) (nodeFilter, error) {
	f := func(n nodeJSON) bool {
		var snap struct {
			Tail []rlp.RawValue `rlp:"tail"`
		}
		return n.N.Load(enr.WithEntry("snap", &snap)) == nil
	}
	return f, nil
}
