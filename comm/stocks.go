package comm

import (
	"fmt"
	"strconv"
	"strings"
	"github.com/kniren/gota/series"
    //"github.com/datochan/gcom/logger"
	"github.com/kniren/gota/dataframe"

	"github.com/datochan/gcom/utils"
)

// 单例避免重复IO操作
var gCalendar *StockCalendar


type CalendarModel struct {
	Date		int
	Open		bool
	prevDate    int
	WeekEnd		bool
	MonthEnd	bool
	QuarterEnd	bool
	YearEnd		bool
}

func NewCalendarModel(date int, open bool, prevDate int, weekEnd, monthEnd, quarterEnd, yearEnd bool) CalendarModel{
	return CalendarModel{date, open, prevDate, weekEnd,
		monthEnd, quarterEnd, yearEnd}
}

type StockCalendar struct {
	calendarDF		dataframe.DataFrame
}

func DefaultStockCalendar(calendarPath string) (*StockCalendar, error){
	if nil != gCalendar {
		return gCalendar, nil
	}
	if 0 >= len(calendarPath) {
		// 需要指定股票日历文件的路径
		return nil, fmt.Errorf("请指定股票日历文件的路径")
	}

	gCalendar = new(StockCalendar)
	err := gCalendar.loadCalendar(calendarPath)
	if nil != err {
		gCalendar = nil
		return nil, fmt.Errorf("载入日历数据出错, 信息为:%s", err.Error())
	}

	return gCalendar, nil
}

func (cal *StockCalendar)Each(f func(dateItem CalendarModel) error) error {
	for _, row := range cal.calendarDF.Maps() {
		prevDate := 0
		if row["prevTradeDate"] != nil {
			prevDate = row["prevTradeDate"].(int)
		}
		dateItem := NewCalendarModel(row["calendarDate"].(int), row["isOpen"].(bool), prevDate,
			row["isWeekEnd"].(bool), row["isMonthEnd"].(bool), row["isQuarterEnd"].(bool), row["isYearEnd"].(bool))
		err := f(dateItem)
		if nil != err {
			return err
		}
	}

	return nil
}

/**
 * 获取指定日期的下一个交易日
 * day: yyyymmdd
 */
func (cal *StockCalendar)NextDay(day string) (string, error) {
	nDay, err := strconv.Atoi(day)
	if nil != err {return "", err}

	filterDF := gCalendar.calendarDF.Filter(dataframe.F{"calendarDate", series.Greater, nDay})

	for _, row := range filterDF.Maps() {
		if true == row["isOpen"] {
			return fmt.Sprintf("%d", row["calendarDate"].(int)), nil
		}
	}
	err = fmt.Errorf("指定日期不存在或者没有下一个交易日")
	return "", err
}

/**
 * 获取指定日期的上一个交易日
 * day: yyyymmdd
 */
func (cal *StockCalendar)PrevDay(day string) (string, error) {
	nDay, err := strconv.Atoi(day)
	if nil != err {return "", err}

	filterDF := gCalendar.calendarDF.Filter(dataframe.F{"calendarDate", series.Eq, nDay})

	idx := utils.FindInStringSlice("prevTradeDate", filterDF.Names())

	return filterDF.Elem(0, idx).String(), nil
}


func (cal *StockCalendar) loadCalendar(calendarPath string) error{
	colTypes := map[string]series.Type{
		"calendarDate": series.Int, "isOpen": series.Bool, "prevTradeDate": series.Int, "isWeekEnd": series.Bool,
		"isMonthEnd": series.Bool, "isQuarterEnd": series.Bool, "isYearEnd": series.Bool}

	cal.calendarDF = utils.ReadCSV(calendarPath, dataframe.WithTypes(colTypes))

	return cal.calendarDF.Err
}

// # 深交所股票代码规则
//新证券代码编码规则升位后的证券代码采用6位数字编码，编码规则定义如下：顺序编码区：6位代码中的第3位到第6位，取值范围为0001-9999。
//证券种类标识区：6位代码中的最左两位，其中第1位标识证券大类，第2位标识该大类下的衍生证券。
//第1位、第2位、第3-6位，定义00xxxx A股证券，03xxxx A股A2权证，07xxxx A股增发，08xxxx A股A1权证，09xxxx A股转配，10xxxx 国债现货，
//11xxxx 债券，12xxxx 可转换债券，13xxxx 国债回购, 150XXX是深市分级基金, 17xxxx 原有投资基金，18xxxx 证券投资基金，20xxxx B股证券，27xxxx B股增发，
//28xxxx B股权证，30xxxx 创业板证券，37xxxx 创业板增发，38xxxx 创业板权证，39xxxx 综合指数/成份指数。
//
//上交所股票代码规则
//在上海证券交易所上市的证券，根据上交所“证券编码实施方案”，采用6位数编制方法，前3位数为区别证券品种，具体见下表所列：
//001×××国债现货；110×××120×××企业债券；129×××100×××可转换债券；201×××国债回购；310×××国债期货；500×××510×××基金；
//600×××A股；700×××配股；710×××转配股；701×××转配股再配股；711×××转配股再转配股；720×××红利；730×××新股申购；
//735×××新基金申购；737×××新股配售；900×××B股。
//
//sh:
//000xxx  指数
//019xxx  上海债券
//11xxxx  上海债券
//12xxxx  上海债券
//13xxxx  上海债券
//14xxxx  上海债券
//50xxxx  基金
//51xxxx  基金
//60xxxx  A股个股
//900xxx  B股个股
//
//
//sz:
//00xxxx  A股个股
//111xxx  债券
//120xxx  债券
//150xxx  分级基金
//159xxx  基金
//16xxxx  基金
//18xxxx  基金
//200xxx  B股个股
//30xxxx  创业板
//399xxx  指数

const (
	STOCKA  = iota   // 股票
	STOCKB           // B股个股
	FUNDS            // 基金
	INDEX            // 指数
	BOND             // 债券
    INDUSTRY         // 行业指数 ...
)

/**
 * 获取股票、基金、指数、行业等信息
 */
func GetFinanceDataFrame(conf IConfigure, types ...int) dataframe.DataFrame{
	stocksPath := fmt.Sprintf("%s%s", conf.GetApp().DataPath, conf.GetTdx().Files.StockList)
	colTypes := map[string]series.Type{
		"code": series.String, "name": series.String, "market": series.Int,
		"unknown1": series.Int, "unknown2": series.Int, "unknown3": series.Int,
		"price": series.Float, "bonus1": series.Int, "bonus2": series.Int}

	baseDF := utils.ReadCSV(stocksPath, dataframe.WithTypes(colTypes))

	var recordIdx []int
	if nil != baseDF.Err { return baseDF }

	for idx, item := range baseDF.Maps() {
		// 行业板块
		if 0 <= utils.FindInIntegerSlice(INDUSTRY, types) {
			if 0 == strings.Index(item["code"].(string), "88") { recordIdx = append(recordIdx, idx) }
		}

		// A股个股
		if 0 <= utils.FindInIntegerSlice(STOCKA, types) {
			if 0 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "00") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "30") { recordIdx = append(recordIdx, idx) }
			}

			if 1 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "6") { recordIdx = append(recordIdx, idx) }
			}
		}

		// B股个股
		if 0 <= utils.FindInIntegerSlice(STOCKB, types) {
			if 0 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "200") { recordIdx = append(recordIdx, idx) }
			}

			if 1 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "900") { recordIdx = append(recordIdx, idx) }
			}
		}

		// 基金
		if 0 <= utils.FindInIntegerSlice(FUNDS, types) {
			if 0 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "15") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "16") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "18") { recordIdx = append(recordIdx, idx) }
			}

			if 1 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "50") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "51") { recordIdx = append(recordIdx, idx) }
			}
		}

		// 指数
		if 0 <= utils.FindInIntegerSlice(INDEX, types) {
			if 0 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "399") { recordIdx = append(recordIdx, idx) }
			}

			if 1 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "000") { recordIdx = append(recordIdx, idx) }
			}
		}

		// 债券(todo: 债券代码待完善和补全, 不建议使用)
		if 0 <= utils.FindInIntegerSlice(BOND, types) {
			if 0 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "111") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "120") { recordIdx = append(recordIdx, idx) }
			}

			if 1 == item["market"] {
				if 0 == strings.Index(item["code"].(string), "019") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "11") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "12") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "13") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "14") { recordIdx = append(recordIdx, idx) }
				if 0 == strings.Index(item["code"].(string), "52") { recordIdx = append(recordIdx, idx) }
			}
		}
	}
	return baseDF.Subset(recordIdx)
}

