package check

import (
	"testing"

	"github.com/hlindberg/tiler/testutils"
)

func Test_Loglevel_valid(t *testing.T) {
	testutils.CheckTrue(Loglevel("info"), t)
	testutils.CheckTrue(Loglevel("debug"), t)
	testutils.CheckTrue(Loglevel("warn"), t)
	testutils.CheckTrue(Loglevel("error"), t)
}

func Test_Loglevel_invalid(t *testing.T) {
	testutils.CheckFalse(Loglevel(" "), t)
	testutils.CheckFalse(Loglevel("what"), t)
	testutils.CheckFalse(Loglevel(""), t)
	testutils.CheckFalse(Loglevel("1"), t)
}
