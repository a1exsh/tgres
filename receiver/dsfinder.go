//
// Copyright 2016 Gregory Trubetskoy. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package receiver manages the receiving end of the data. All of the
// queueing, caching, perioding flushing and cluster forwarding logic
// is here.
package receiver

import (
	"time"

	"github.com/tgres/tgres/rrd"
)

type dftDSFinder struct{}

type MatchingDSSpecFinder interface {
	FindMatchingDSSpec(name string) *rrd.DSSpec
}

func (_ *dftDSFinder) FindMatchingDSSpec(name string) *rrd.DSSpec {
	if name == "" {
		return nil
	}
	return &rrd.DSSpec{
		Step:      10 * time.Second,
		Heartbeat: 2 * time.Hour,
		RRAs: []rrd.RRASpec{
			rrd.RRASpec{Function: rrd.WMEAN,
				Step: 10 * time.Second,
				Span: 6 * time.Hour,
			},
			rrd.RRASpec{Function: rrd.WMEAN,
				Step: 1 * time.Minute,
				Span: 24 * time.Hour,
			},
			rrd.RRASpec{Function: rrd.WMEAN,
				Step: 10 * time.Minute,
				Span: 93 * 24 * time.Hour,
			},
			rrd.RRASpec{Function: rrd.WMEAN,
				Step: 24 * time.Hour,
				Span: 1825 * 24 * time.Hour,
			},
		},
	}
}
