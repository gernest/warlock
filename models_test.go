package warlock

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	Convey("Configuration", t, func() {
		Convey("Default", func() {
			c := NewConfig(nil)

			So(c.DB, ShouldEqual, "warlock.db")
			So(c.SessMaxAge, ShouldEqual, 30)
		})
		Convey("Custom", func() {
			c := &Config{DB: "chaos.db"}
			cfg := NewConfig(c)

			cc := new(Config)
			cc.DB = "chaos.db"
			ccf := NewConfig(cc)

			So(cfg.DB, ShouldEqual, "chaos.db")
			So(ccf.DB, ShouldEqual, "chaos.db")
			So(ccf.SessMaxAge, ShouldEqual, 30)

		})
	})
}
