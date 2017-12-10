package ctdx

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/datochan/gcom/utils"
	"github.com/datochan/ctdx/comm"

)

func TestReportList(t *testing.T) {
	configure := new(comm.Conf)
	configure.Parse("/Users/datochan/WorkSpace/GoglandProjects/src/ctdx/configure.toml")

	Convey("检测获取指定年份和股票的财报", t, func() {
		// 默认加载股票日历数据
		df := ReportList(configure, "600000", "20170930")
		code := utils.Element(df, 0, "code")
		date := utils.Element(df, 0, "date")
		price1 := utils.Element(df, 0, "1")

		So(code.String(), ShouldEqual, "600000")
		So(date.String(), ShouldEqual, "20170930")
		So(price1.Float(), ShouldEqual, 1.45)
	})

	Convey("检测获取指定股票的历年财报", t, func() {
		// 默认加载股票日历数据
		df := ReportList(configure, "600000", "")

		code := utils.Element(df, 0, "code")
		date1 := utils.Element(df, 0, "date")
		date2 := utils.Element(df, 1, "date")
		date3 := utils.Element(df, 2, "date")
		price1 := utils.Element(df, 0, "1")
		price2 := utils.Element(df, 1, "1")
		price3 := utils.Element(df, 3, "1")

		So(code.String(), ShouldEqual, "600000")

		So(date1.String(), ShouldEqual, "19961231")
		So(date2.String(), ShouldEqual, "19971231")
		So(date3.String(), ShouldEqual, "19971231")

		So(price1.Float(), ShouldEqual, 0.63)
		So(price2.Float(), ShouldEqual, 0.32)
		So(price3.Float(), ShouldEqual, 0.43)
	})
}