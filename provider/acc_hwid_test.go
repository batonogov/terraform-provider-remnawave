package provider

import (
	"testing"
)

// HWID endpoints return 404/500 in the test environment — the HWID feature
// requires specific panel configuration that is not available in the
// Docker-based acceptance test panel. These tests are skipped.

func TestAccHwidDeviceResource(t *testing.T) {
	t.Skip("HWID device CRUD requires panel HWID feature enabled — not available in test env")
}

func TestAccHwidStatsDataSource(t *testing.T) {
	t.Skip("HWID stats requires panel HWID feature enabled — not available in test env")
}

func TestAccHwidTopUsersDataSource(t *testing.T) {
	t.Skip("HWID top users requires panel HWID feature enabled — not available in test env")
}
