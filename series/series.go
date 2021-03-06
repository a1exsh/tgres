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

// Package series provides fundamental series operations.
package series

import "time"

type Series interface {
	Next() bool
	Close() error

	CurrentValue() float64

	// The time one which the current slot ends. The next slot begins
	// immediately after this time.
	CurrentTime() time.Time

	Step() time.Duration
	GroupBy(...time.Duration) time.Duration
	TimeRange(...time.Time) (time.Time, time.Time)
	Latest() time.Time
	MaxPoints(...int64) int64

	//Alias(...string) string
}
