//
// Copyright 2015 Gregory Trubetskoy. All Rights Reserved.
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

package daemon

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/tgres/tgres/misc"
	"github.com/tgres/tgres/rrd"
)

type Config struct { // Needs to be exported for TOML to work
	PidPath                  string   `toml:"pid-file"`
	LogPath                  string   `toml:"log-file"`
	LogCycle                 duration `toml:"log-cycle-interval"`
	DbConnectString          string   `toml:"db-connect-string"`
	MaxCachedPoints          int      `toml:"max-cached-points"`
	MaxCache                 duration `toml:"max-cache-duration"`
	MinCache                 duration `toml:"min-cache-duration"`
	MaxFlushesPerSecond      int      `toml:"max-flushes-per-second"`
	GraphiteTextListenSpec   string   `toml:"graphite-text-listen-spec"`
	GraphiteUdpListenSpec    string   `toml:"graphite-udp-listen-spec"`
	GraphitePickleListenSpec string   `toml:"graphite-pickle-listen-spec"`
	StatsdTextListenSpec     string   `toml:"statsd-text-listen-spec"`
	StatsdUdpListenSpec      string   `toml:"statsd-udp-listen-spec"`
	HttpListenSpec           string   `toml:"http-listen-spec"`
	Workers                  int
	DSs                      []ConfigDSSpec `toml:"ds"`
	StatFlush                duration       `toml:"stat-flush-interval"`
	StatsNamePrefix          string         `toml:"stats-name-prefix"`
}

type regex struct{ *regexp.Regexp }

func (r *regex) UnmarshalText(text []byte) (err error) {
	r.Regexp, err = regexp.Compile(string(text))
	return err
}

type duration struct{ time.Duration }

func (d *duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// Needs to be exported for TOML
type ConfigDSSpec struct {
	Regexp    regex
	Step      duration
	Heartbeat duration
	RRAs      []ConfigRRASpec
}
type ConfigRRASpec struct {
	Function rrd.Consolidation
	Step     time.Duration
	Span     time.Duration
	Xff      float64
}

func (r *ConfigRRASpec) UnmarshalText(text []byte) error {
	r.Xff = 0.5
	parts := strings.SplitN(string(text), ":", 4)
	if len(parts) < 2 || len(parts) > 4 {
		return fmt.Errorf("Invalid RRA specification (not enough or too many elements): %q", string(text))
	}

	// If first character of first part is a digit, assume we're
	// skipping CF and default to WMEAN.
	if len(parts[0]) > 0 && strings.Contains("0123456789", string(parts[0][0])) {
		parts = append([]string{"WMEAN"}, parts...)
	}

	switch strings.ToUpper(parts[0]) {
	case "WMEAN":
		r.Function = rrd.WMEAN
	case "MIN":
		r.Function = rrd.MIN
	case "MAX":
		r.Function = rrd.MAX
	case "LAST":
		r.Function = rrd.LAST
	default:
		return fmt.Errorf("Invalid consolidation: %q (valid funcs: wmean, min, max, last)", parts[0])
	}

	var err error
	if r.Step, err = misc.BetterParseDuration(parts[1]); err != nil {
		return fmt.Errorf("Invalid Step: %q (%v)", parts[1], err)
	}
	if r.Span, err = misc.BetterParseDuration(parts[2]); err != nil {
		return fmt.Errorf("Invalid Size: %q (%v)", parts[2], err)
	}
	if (r.Span.Nanoseconds() % r.Step.Nanoseconds()) != 0 {
		newSpan := time.Duration(r.Span.Nanoseconds()/r.Step.Nanoseconds()*r.Step.Nanoseconds()) * time.Nanosecond
		log.Printf("Span (%q) is not a multiple of step (%q), auto adjusting span to %v.", parts[2], parts[1], newSpan)
		r.Span = newSpan
		if newSpan.Nanoseconds() == 0 {
			return fmt.Errorf("invalid Size (%v)", newSpan)
		}
	}
	if len(parts) == 4 {
		var err error
		if r.Xff, err = strconv.ParseFloat(parts[3], 64); err != nil {
			return fmt.Errorf("Invalid XFF: %q (%v)", parts[3], err)
		}
	}
	return nil
}

var readConfig = func(cfgPath string) (*Config, error) {
	cfg := &Config{}
	_, err := toml.DecodeFile(cfgPath, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) processConfigPidFile(wd string) error {
	if c.PidPath == "" {
		return fmt.Errorf("pid-file setting empty")
	}
	if !filepath.IsAbs(c.PidPath) {
		if wd == "" {
			return fmt.Errorf("pid-file must be absolute path if working directory cannot be determined")
		}
		c.PidPath = filepath.Join(wd, c.PidPath)
	}
	pidDir, _ := filepath.Split(c.PidPath)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return errors.New(fmt.Sprintf("Unable to create directory: '%s' (%v).", pidDir, err))
	}
	return nil
}

func (c *Config) processConfigLogFile(wd string) error {
	if os.Getenv("TGRES_LOG") != "" {
		c.LogPath = os.Getenv("TGRES_LOG")
	}
	if c.LogPath == "" {
		return fmt.Errorf("log-file setting empty")
	}
	if !filepath.IsAbs(c.LogPath) {
		if wd == "" {
			return fmt.Errorf("log-file must be absolute path if working directory cannot be determined")
		}
		c.LogPath = filepath.Join(wd, c.LogPath)
	}
	logDir, _ := filepath.Split(c.LogPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return errors.New(fmt.Sprintf("Unable to create directory: '%s' (%v).", logDir, err))
	}

	log.Printf("Logs will be written to '%s'.", c.LogPath)
	return nil
}

func (c *Config) processConfigLogCycleInterval() error {
	if c.LogCycle.Duration == 0 {
		return fmt.Errorf("log-cycle-interval setting empty")
	}
	log.Printf("Will cycle logs every %v (log-cycle-interval).", c.LogCycle.Duration)

	logDir, _ := filepath.Split(c.LogPath)
	log.Printf("All further status messages will be written to log file(s) in '%s'.", logDir)
	logFileCycler(c.LogPath, c.LogCycle.Duration)
	log.Print("Server starting.")

	return nil
}

func (c *Config) processDbConnectString() error {
	if os.Getenv("TGRES_DB_CONNECT") != "" {
		c.DbConnectString = os.Getenv("TGRES_DB_CONNECT")
	}
	if c.DbConnectString == "" {
		return fmt.Errorf("db-connect-string empty")
	}
	return nil
}

func (c *Config) processMaxCachedPoints() error {
	if c.MaxCachedPoints == 0 {
		return fmt.Errorf("max-cached-points missing, must be integer")
	}
	log.Printf("Data Sources will be flushed when cached points exceeds %d (max-cached-points).", c.MaxCachedPoints)
	return nil
}

func (c *Config) processMaxCacheDuration() error {
	if c.MaxCache.Duration == 0 {
		return fmt.Errorf("max-cache-duration is missing")
	} else {
		log.Printf("Data Sources will be flushed after (approximately) %v (max-cache-duration).", c.MaxCache.Duration)
	}
	return nil
}

func (c *Config) processMinCacheDuration() error {
	if c.MinCache.Duration == 0 {
		return fmt.Errorf("min-cache-duration is missing")
	} else if c.MinCache.Duration > c.MaxCache.Duration/2 {
		return fmt.Errorf("max-cache-duration should be at least twice min-cache-duration")
	} else {
		log.Printf("A Data Source will be flushed at most once per %v (min-cache-duration).", c.MinCache.Duration)
	}
	return nil
}

func (c *Config) processMaxFlushesPerSecond() error {
	if c.MaxFlushesPerSecond == 0 {
		return fmt.Errorf("max-flushes-per-second missing, must be integer")
	}
	log.Printf("Data Source flushes will be rate limited to %d per second (max-flushes-per-second).", c.MaxFlushesPerSecond)
	return nil
}

func (c *Config) processStatFlushInterval() error {
	if c.StatFlush.Duration == 0 {
		return fmt.Errorf("stat-flush-interval is missing")
	} else {
		log.Printf("Stats (a la statsd) will be flushed every %v (stat-flush-interval).", c.StatFlush.Duration)
	}
	return nil
}

func (c *Config) processStatsNamePrefix() error {
	if c.StatsNamePrefix == "" {
		log.Printf("stats-name-prefix is empty, defaulting to 'stats'")
		c.StatsNamePrefix = "stats"

	}
	return nil
}

func (c *Config) processWorkers() error {
	if c.Workers == 0 {
		return fmt.Errorf("workers missing, must be an integer")
	}
	log.Printf("Number of workers (and flushers) will be %d.", c.Workers)
	return nil
}

func (c *Config) processDSSpec() error {
	// TODO validate function, regular expression, all that
	for _, ds := range c.DSs {
		for _, rra := range ds.RRAs {
			if (rra.Step.Nanoseconds() % ds.Step.Duration.Nanoseconds()) != 0 {
				newStep := time.Duration(rra.Step.Nanoseconds()/ds.Step.Duration.Nanoseconds()*ds.Step.Duration.Nanoseconds()) * time.Nanosecond
				log.Printf("DS %q: RRA step (%v) is not a multiple of DS Step (%v), auto adjusting Step to %v.", ds.Regexp.String(), rra.Step, ds.Step.Duration, newStep)
				if newStep.Nanoseconds() == 0 {
					return fmt.Errorf("DS %q: invalid Step (%v)", ds.Regexp.String(), newStep)
				}
				rra.Step = newStep
			}
		}
	}
	// TODO xff?
	return nil
}

func (c *Config) FindMatchingDSSpec(name string) *rrd.DSSpec {
	for _, dsSpec := range c.DSs {
		if dsSpec.Regexp.Regexp.MatchString(name) {
			return convertDSSpec(&dsSpec)
		}
	}
	return nil
}

func convertDSSpec(dsSpec *ConfigDSSpec) *rrd.DSSpec {
	serdeDSSpec := &rrd.DSSpec{
		Step:      dsSpec.Step.Duration,
		Heartbeat: dsSpec.Heartbeat.Duration,
		RRAs:      make([]rrd.RRASpec, len(dsSpec.RRAs)),
	}
	for i, r := range dsSpec.RRAs {
		serdeDSSpec.RRAs[i] = rrd.RRASpec{
			Function: r.Function,
			Step:     r.Step,
			Span:     r.Span,
			Xff:      float32(r.Xff),
		}
	}
	return serdeDSSpec
}

type configer interface {
	processConfigPidFile(string) error
	processConfigLogFile(string) error
	processConfigLogCycleInterval() error
	processDbConnectString() error
	processMaxCachedPoints() error
	processMaxCacheDuration() error
	processMinCacheDuration() error
	processMaxFlushesPerSecond() error
	processStatFlushInterval() error
	processStatsNamePrefix() error
	processWorkers() error
	processDSSpec() error
}

var processConfig = func(c configer, wd string) error {

	if err := c.processConfigPidFile(wd); err != nil {
		return err
	}
	if err := c.processConfigLogFile(wd); err != nil {
		return err
	}
	if err := c.processConfigLogCycleInterval(); err != nil {
		return err
	}
	if err := c.processDbConnectString(); err != nil {
		return err
	}
	if err := c.processMaxCachedPoints(); err != nil {
		return err
	}
	if err := c.processMaxCacheDuration(); err != nil {
		return err
	}
	if err := c.processMinCacheDuration(); err != nil {
		return err
	}
	if err := c.processMaxFlushesPerSecond(); err != nil {
		return err
	}
	if err := c.processStatFlushInterval(); err != nil {
		return err
	}
	if err := c.processStatsNamePrefix(); err != nil {
		return err
	}
	if err := c.processWorkers(); err != nil {
		return err
	}
	if err := c.processDSSpec(); err != nil {
		return err
	}
	return nil
}
