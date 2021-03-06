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

package dsl

import (
	"time"

	"github.com/tgres/tgres/rrd"
	"github.com/tgres/tgres/serde"
	"github.com/tgres/tgres/series"
)

type rcacheSeriesQuerier interface {
	serde.DataSourceNamesFetcher
	serde.DataSourceFetcher
	serde.SeriesQuerier
}

type ReadCache struct {
	rcacheSeriesQuerier
	dsns *DataSourceNames
}

func NewReadCache(db rcacheSeriesQuerier) *ReadCache {
	return &ReadCache{rcacheSeriesQuerier: db, dsns: &DataSourceNames{}}
}

func (r *ReadCache) dsIdsFromIdent(ident string) map[string]int64 {
	result := r.dsns.dsIdsFromIdent(ident)
	if len(result) == 0 {
		r.dsns.reload(r)
		result = r.dsns.dsIdsFromIdent(ident)
	}
	return result
}

func (r *ReadCache) FsFind(pattern string) []*FsFindNode {
	r.dsns.reload(r)
	return r.dsns.fsFind(pattern)
}

func NewReadCacheFromMap(dss map[string]rrd.DataSourcer) *ReadCache {
	return NewReadCache(newMapCache(dss))
}

// A rcacheSeriesQuerier backed by a simple map of DSs
func newMapCache(dss map[string]rrd.DataSourcer) *mapCache {
	mc := &mapCache{make(map[string]int64), make(map[int64]rrd.DataSourcer)}
	var n int64
	for name, ds := range dss {
		mc.byName[name] = n
		mc.byId[n] = ds
		n++
	}
	return mc
}

type mapCache struct {
	byName map[string]int64
	byId   map[int64]rrd.DataSourcer
}

func (m mapCache) FetchDataSourceNames() (map[string]int64, error) {
	return m.byName, nil
}

func (m *mapCache) FetchDataSourceById(id int64) (rrd.DataSourcer, error) {
	return m.byId[id], nil
}

func (*mapCache) SeriesQuery(ds rrd.DataSourcer, from, to time.Time, maxPoints int64) (series.Series, error) {
	return series.NewRRASeries(ds.RRAs()[0]), nil
}

func (m *mapCache) FetchDataSources() ([]rrd.DataSourcer, error) {
	result := []rrd.DataSourcer{}
	for _, ds := range m.byId {
		result = append(result, ds)
	}
	return result, nil
}
