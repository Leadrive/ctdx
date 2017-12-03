package ctdx

import (
	"os"
	"fmt"
	"bytes"
	"encoding/hex"
	"encoding/binary"
	"github.com/kniren/gota/dataframe"

	gbytes "github.com/datochan/gcom/bytes"
	"github.com/datochan/gcom/utils"
	"github.com/datochan/gcom/logger"

	pkg "github.com/datochan/ctdx/packet"
	"github.com/datochan/gcom/cnet"
)

func UnknownPkgHandler(session cnet.ISession, packet interface{}) {
	respNode := packet.(pkg.ResponseNode)
	switch respNode.EventId {
	case 0x0B: logger.Info("模拟设备已注册成功！")
	case 0x0FDB:
		noticeByte := respNode.RawData.([]byte)
		noticeTmp := []rune(utils.ConvertTo(gbytes.BytesToString(noticeByte[0xb2:]), "gbk", "utf8"))
		logger.Info("收到代理服务器的公告信息:%s ...", string(noticeTmp[:70]))

	default:
		logger.Info("收到未知封包:%s", hex.EncodeToString(respNode.RawData.([]byte)))
	}
}


/**
 * 接收市场行情的初始数据
 */
func (client *TdxClient) OnMarketInitInfo(session cnet.ISession, packet interface{}){
	var newBuffer bytes.Buffer
	var notice pkg.MarketInitInfo

	respNode := packet.(pkg.ResponseNode)

	newBuffer.Write(respNode.RawData.([]byte))
	err := binary.Read(&newBuffer, binary.LittleEndian, &notice)

	if nil != err { return }

	client.lastTrade.ServerName = utils.ConvertTo(gbytes.BytesToString(notice.ServerName[:]), "gbk", "utf8")
	client.lastTrade.Domain = gbytes.BytesToString(notice.DomainUrl[:])
	client.lastTrade.SHDate = notice.DateSH
	client.lastTrade.SHFlag = notice.LastSHFlag

	client.lastTrade.SZDate = notice.DateSZ
	client.lastTrade.SZFlag = notice.LastSZFlag

	logger.Info("市场最新交易信息: 券商名称:%s, 最后交易时间:%d", client.lastTrade.ServerName, client.lastTrade.SZDate)
}

/**
 * 接收市场股票数量信息
 */
func (client *TdxClient) OnStockCount(session cnet.ISession, packet interface{}){
	var newBuffer bytes.Buffer
	var stockCount uint16
	respNode := packet.(pkg.ResponseNode)
	newBuffer.Write(respNode.RawData.([]byte))
	binary.Read(&newBuffer, binary.LittleEndian, &stockCount)

	if respNode.CmdId == 0x6B { client.lastTrade.SZCount = uint32(stockCount) } // 深圳股票数量
	if respNode.CmdId == 0x6C { client.lastTrade.SHCount = uint32(stockCount) } // 上海股票数量
}

/**
 * 获取股票基础信息
 */
func (client *TdxClient) OnStockBase(session cnet.ISession, packet interface{}){
	var newBuffer bytes.Buffer
	var stockItem pkg.StockBaseItem
	var stockList []StockBaseModel

	market := 0

	itemSize := utils.SizeStruct(pkg.StockBaseItem{})
	respNode := packet.(pkg.ResponseNode)

	if respNode.CmdId == 0x6E { market = 1 }

	littleEndianBuffer := gbytes.NewLittleEndianStream(respNode.RawData.([]byte))

	stockCount, _ := littleEndianBuffer.ReadUint16()  // 读取股票数量

	for idx :=0; idx < int(stockCount); idx++ {
		tmpBuffer, _ := littleEndianBuffer.ReadBuff(itemSize)
		newBuffer.Write(tmpBuffer)

		binary.Read(&newBuffer, binary.LittleEndian, &stockItem)

		stockModel := StockBaseModel{gbytes.BytesToString(stockItem.Code[:]),
		utils.ConvertTo(gbytes.BytesToString(stockItem.Name[:]), "gbk", "utf8"), market,
		int(stockItem.Unknown1), int(stockItem.Unknown2), int(stockItem.Unknown3),
		float64(stockItem.Price), int(stockItem.Bonus1), int(stockItem.Bonus2)}

		stockList = append(stockList, stockModel)
	}

	stockBaseDF := dataframe.LoadStructs(stockList)

	if nil != stockBaseDF.Err {
		logger.Error("加载新的股票数据时发生错误: %v", stockBaseDF.Err)
		return
	}

	if 0 >= client.stockBaseDF.Nrow() {
		client.stockBaseDF = stockBaseDF
	} else {
		client.stockBaseDF = client.stockBaseDF.RBind(stockBaseDF)
	}

	if client.stockBaseDF.Nrow() >= int(client.lastTrade.SZCount + client.lastTrade.SHCount) {
		// 更新结束
		client.stockBaseDF.SetNames("code", "name", "market", "unknown1", "unknown2", "unknown3", "price", "bonus1", "bonus2")
		stockBasePath := fmt.Sprintf("%s%s", client.Configure.GetApp().DataPath, client.Configure.GetTdx().Files.StockList)
		utils.WriteCSV(stockBasePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, &client.stockBaseDF)

		client.dispatcher.DelHandler(uint32(respNode.EventId))

		client.Finished <- nil
	}
}

/**
 * 获取股票权息数据
 */
func (client *TdxClient) OnStockBonus(session cnet.ISession, packet interface{}){
	var newBuffer bytes.Buffer
	var bonusItem pkg.StockBonusItem
	var bonusList []StockBonusModel

	respNode := packet.(pkg.ResponseNode)
	itemSize := utils.SizeStruct(pkg.StockBonusItem{})
	littleEndianBuffer := gbytes.NewLittleEndianStream(respNode.RawData.([]byte))

	stockCount, _ := littleEndianBuffer.ReadUint16()  // 读取股票数量

	logger.Info("\t收到 %d 只股票的权息数据...", stockCount)

	for stockIdx :=0; stockIdx < int(stockCount); stockIdx++ {
		littleEndianBuffer.ReadBuff(7)  // 跳过股票代码与市场标识
		bonusCount, _ := littleEndianBuffer.ReadUint16()  // 某只股票的权息条数

		for bonusIdx:=0;bonusIdx<int(bonusCount);bonusIdx++ {
			tmpBuffer, _ := littleEndianBuffer.ReadBuff(itemSize)
			newBuffer.Write(tmpBuffer)

			binary.Read(&newBuffer, binary.LittleEndian, &bonusItem)

			bonusModel := StockBonusModel{ gbytes.BytesToString(bonusItem.Code[:]),
				int(bonusItem.Date),
				int(bonusItem.Market),int(bonusItem.Type),
				float64(bonusItem.Money),float64(bonusItem.Price), float64(bonusItem.Count),
				float64(bonusItem.Rate)}

			bonusList = append(bonusList, bonusModel)
		}
	}

	// 更新结束
	bonusDF := dataframe.LoadStructs(bonusList)
	if nil != bonusDF.Err {
		logger.Error(fmt.Sprintf("加载权息数据时发生错误:%v", bonusDF.Err))
		return
	}

	if 0 >= client.stockbonusDF.Nrow() {
		client.stockbonusDF = bonusDF
	} else {
		client.stockbonusDF = client.stockbonusDF.RBind(bonusDF)
	}

	if stockBonusFinishedIdx != respNode.Index {
		client.bonusFinishedChan <- int(stockCount)
		return
	}

	client.dispatcher.DelHandler(uint32(respNode.EventId))

	client.stockbonusDF.SetNames("code", "date", "market", "type", "money", "price", "count", "rate")
	calendarPath := fmt.Sprintf("%s%s", client.Configure.GetApp().DataPath, client.Configure.GetTdx().Files.StockBonus)
	utils.WriteCSV(calendarPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, &client.stockbonusDF)
	client.dispatcher.DelHandler(uint32(respNode.EventId))

	client.Finished <- nil
	return
}

func (client *TdxClient) onStockDayHistory(stockLength int, littleEndianBuffer *gbytes.LittleEndianStreamImpl) dataframe.DataFrame {
	var newBuffer bytes.Buffer
	var stockDayItem pkg.StockDayItem
	var stockDaysList []StockDayModel

	itemSize := utils.SizeStruct(pkg.StockMinsItem{})
	stockCount := int(stockLength)/itemSize

	for idx:=0; idx<stockCount; idx++{
		tmpBuffer, _ := littleEndianBuffer.ReadBuff(itemSize)
		newBuffer.Write(tmpBuffer)

		binary.Read(&newBuffer, binary.LittleEndian, &stockDayItem)

		stockDayModel := StockDayModel{int(stockDayItem.Date),
			float64(stockDayItem.Open)/100.0,float64(stockDayItem.Low)/100.0,
			float64(stockDayItem.High)/100.0,float64(stockDayItem.Close)/100.0,
			int(stockDayItem.Volume),float64(stockDayItem.Amount)/100.0}

		stockDaysList = append(stockDaysList, stockDayModel)
	}

	if 0 >= len(stockDaysList) {
		return dataframe.DataFrame{Err: fmt.Errorf("没有任何行情数据")}
	}

	return dataframe.LoadStructs(stockDaysList)
}


func (client *TdxClient) onStockMinsHistory(stockLength int, littleEndianBuffer *gbytes.LittleEndianStreamImpl) dataframe.DataFrame {
	var newBuffer bytes.Buffer
	var stockMinsItem pkg.StockMinsItem
	var stockMinsList []StockMinsModel

	itemSize := utils.SizeStruct(pkg.StockMinsItem{})
	stockCount := int(stockLength)/itemSize

	for idx:=0; idx<stockCount; idx++{
		tmpBuffer, _ := littleEndianBuffer.ReadBuff(itemSize)
		newBuffer.Write(tmpBuffer)

		binary.Read(&newBuffer, binary.LittleEndian, &stockMinsItem)

		nYear := int(stockMinsItem.Date) / 2048 + 2004
		nMonth := int(stockMinsItem.Date % 2048 / 100)
		nDay := int(stockMinsItem.Date % 2048 % 100)

		nDate := nYear*10000 + nMonth*100 + nDay
		strTime := fmt.Sprintf("%02d:%02d:00", int(stockMinsItem.Time)/60, int(stockMinsItem.Time)%60)

		stockMinsModel := StockMinsModel{nDate, strTime,
			float64(stockMinsItem.Open)/100.0,float64(stockMinsItem.Low)/100.0,
			float64(stockMinsItem.High)/100.0,float64(stockMinsItem.Close)/100.0,
			int(stockMinsItem.Volume),float64(stockMinsItem.Amount)/100.0}

		stockMinsList = append(stockMinsList, stockMinsModel)
	}

	if 0 >= len(stockMinsList) {
		return dataframe.DataFrame{Err: fmt.Errorf("没有任何行情数据")}
	}

	return dataframe.LoadStructs(stockMinsList)
}

/**
 * 保存行情数据
 */
func (client *TdxClient) historySaveFile(df dataframe.DataFrame, stocksPath string) {
	isExist, _ := utils.FileExists(stocksPath)
	if ! isExist {
		utils.WriteCSV(stocksPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, &df)
	} else {
		utils.WriteCSV(stocksPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, &df, dataframe.WriteHeader(false))
	}
}

/**
 * 接收行情数据
 */
func (client *TdxClient) OnStockHistory(session cnet.ISession, packet interface{}) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("panic recover! p: %v", p)
		}

	}()

	respNode := packet.(pkg.ResponseNode)

	if 0xffff == respNode.Index {
		// 更新结束
		client.dispatcher.DelHandler(uint32(respNode.EventId))
		client.Finished <- nil
		return
	}

	// 收到盘后行情数据
	littleEndianBuffer := gbytes.NewLittleEndianStream(respNode.RawData.([]byte))

	littleEndianBuffer.ReadUint16()                   // 略过标识符
	stockLength, _ := littleEndianBuffer.ReadUint32() // 读取股票数量

	idx := utils.FindInStringSlice("code", client.stockBaseDF.Names())
	strCode := client.stockBaseDF.Elem(int(respNode.Index-1), idx).String()
	idx = utils.FindInStringSlice("market", client.stockBaseDF.Names())
	market, _ := client.stockBaseDF.Elem(int(respNode.Index-1), idx).Int()

	//logger.Info("\t已收到 %d%s 的盘后行情数据...", market, strCode)

	fileName := fmt.Sprintf("%d%s.csv.zip", market, strCode)

	if respNode.CmdId == pkg.GenerateStockDayItem(0, "", 0, 0, 0).CmdId {
		df := client.onStockDayHistory(int(stockLength), littleEndianBuffer)
		if nil != df.Err {
			//logger.Info("\t接收行情 %d%s 的数据出错, Err: %v", market, strCode, df.Err)
			return
		}

		df.SetNames("date", "open", "low", "high", "close", "volume", "amount")

		stocksPath := fmt.Sprintf("%s%s%s", client.Configure.GetApp().DataPath, client.Configure.GetTdx().Files.StockDay, fileName)
		client.historySaveFile(df, stocksPath)
		return
	}

	df := client.onStockMinsHistory(int(stockLength), littleEndianBuffer)
	if nil != df.Err {
		//logger.Info("\t接收行情 %d%s 的数据出错, Err: %v", market, strCode, df.Err)
		return
	}

	df.SetNames("date", "time", "open", "low", "high", "close", "volume", "amount")

	stocksPath := fmt.Sprintf("%s%s%s", client.Configure.GetApp().DataPath, client.Configure.GetTdx().Files.StockMin, fileName)
	client.historySaveFile(df, stocksPath)
}
