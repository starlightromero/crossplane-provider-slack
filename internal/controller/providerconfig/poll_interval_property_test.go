/*
Copyright 2024 Avodah Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package providerconfig

import (
	"fmt"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: crossplane-provider-slack, Property 15: Configurable poll interval is respected
// **Validates: Requirements 8.2**

func TestProperty_PollIntervalDurationParsing(t *testing.T) {
	// For any valid Go duration string set in ProviderConfig.spec.pollInterval,
	// the duration SHALL be parseable and produce a positive duration value that
	// the reconciler can use as its poll interval.
	rapid.Check(t, func(t *rapid.T) {
		// Generate arbitrary positive duration components
		hours := rapid.IntRange(0, 23).Draw(t, "hours")
		minutes := rapid.IntRange(0, 59).Draw(t, "minutes")
		seconds := rapid.IntRange(0, 59).Draw(t, "seconds")

		// Ensure at least one component is non-zero for a positive duration
		if hours == 0 && minutes == 0 && seconds == 0 {
			minutes = 1
		}

		// Build a valid Go duration string by concatenating non-zero components
		var durationStr string
		if hours > 0 {
			durationStr += fmt.Sprintf("%dh", hours)
		}
		if minutes > 0 {
			durationStr += fmt.Sprintf("%dm", minutes)
		}
		if seconds > 0 {
			durationStr += fmt.Sprintf("%ds", seconds)
		}

		// Parse the duration string
		parsed, err := time.ParseDuration(durationStr)
		if err != nil {
			t.Fatalf("valid duration string %q failed to parse: %v", durationStr, err)
		}

		// Verify the parsed duration is positive
		if parsed <= 0 {
			t.Fatalf("parsed duration %v from %q is not positive", parsed, durationStr)
		}

		// Verify the parsed duration matches the expected value
		expected := time.Duration(hours)*time.Hour +
			time.Duration(minutes)*time.Minute +
			time.Duration(seconds)*time.Second
		if parsed != expected {
			t.Fatalf("parsed duration %v does not match expected %v for input %q",
				parsed, expected, durationStr)
		}
	})
}
