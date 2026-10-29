//go:build !race

package api_tests

// raceEnabled is whether tests are running with the race detector, which makes code several times slower.
const raceEnabled = false
