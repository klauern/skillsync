package e2e_test

import (
	"flag"
	"os"
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestMain(m *testing.M) {
	flag.Parse()
	e2e.SetUpdateGolden(*updateGolden)
	os.Exit(m.Run())
}
