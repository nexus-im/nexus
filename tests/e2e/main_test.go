package e2e

import (
	"os"
	"testing"

	"github.com/nexus-im/nexus/tests/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}
