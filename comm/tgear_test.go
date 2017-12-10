package comm

import (
	"testing"
	"encoding/hex"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBufferToDouble(t *testing.T) {
	var index int
	var resultList []float64

	Convey("测试解密成double类型", t, func() {
		buffer, _ := hex.DecodeString("AC0E504C246180FDA60EEC0EA0BC4D978C0204880F4EB7901EA9AB2F019C8E044100")

		for index < len(buffer) {
			len, result := BufferToDouble(buffer[index:])
			resultList = append(resultList, result)
			index += len
		}

		So(resultList[0], ShouldEqual, 940.0)
		So(resultList[1], ShouldEqual, -16.0)
		So(resultList[2], ShouldEqual, -12.0)
		So(resultList[5], ShouldEqual, 14999360.0)
	})
}