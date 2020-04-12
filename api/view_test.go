package api

import (
	"testing"

	"github.com/labstack/echo"
)

var testSrv *echo.Echo

func testSetup(tb testing.TB) {
	testSrv = echo.New()

}

func testTeardown(tb testing.TB) {

}
