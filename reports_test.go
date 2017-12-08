package ctdx

import (
	"testing"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/datochan/gcom/utils"
	"github.com/datochan/ctdx/comm"
)

func TestReportList(t *testing.T) {
	configure := new(comm.Conf)
	configure.Parse("/Users/datochan/WorkSpace/GoglandProjects/src/ctdx/configure.toml")

	Convey("检测获取财报", t, func() {
		// 默认加载股票日历数据
		df := ReportList(configure, "", "20170930")
		codeIdx := utils.FindInStringSlice("code", df.Names())
		dateIdx := utils.FindInStringSlice("date", df.Names())
		idx1 := utils.FindInStringSlice("1", df.Names())
		idx2 := utils.FindInStringSlice("2", df.Names())
		idx3 := utils.FindInStringSlice("3", df.Names())
		idx4 := utils.FindInStringSlice("4", df.Names())
		idx5 := utils.FindInStringSlice("5", df.Names())

		subDF := df.Select([]int{dateIdx, codeIdx, idx1, idx2, idx3, idx4, idx5})

		fmt.Println(subDF)
	})
}