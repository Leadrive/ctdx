package comm

/**
 * 从tdx的封包中解密价格数据
 */
func BufferToDouble(buffer []byte) (int, float64) {
	idx := 0
	deltaValue := 64.0
	doubleValue := float64(buffer[0] & 0x3F)

	for buffer[idx] & 0x80 != 0 {
		idx += 1
		doubleValue += float64(buffer[idx] & 0x7F) * deltaValue
		deltaValue *= 128.0
	}

	if 0 != buffer[0] & 0x40 {
		doubleValue = -doubleValue
	}

	return idx+1, doubleValue
}

// todo: 待处理
func DoubleToBuf(inputValue float64, lpTargetBuffer interface{}) {
	//signed __int16 flag; // bx@1 数值的正负标记，负数为1，正数为0
	//int idx; // esi@1
	//signed __int64 deltaValue; // qax@2
	//unsigned int v5; // eax@4
	//BYTE v6; // cl@4
	//int bufferLength; // eax@7
	//LPVOID v8; // ecx@8
	//BYTE v9; // dl@9
	//
	//idx = 0;
	//flag = 0;
	//if ( inputValue >= 0.0 ) {
	//	deltaValue = (signed __int64)(inputValue + 0.5);
	//} else {
	//	deltaValue = (signed __int64)(0.5 - inputValue);
	//	flag = 1;
	//}
	//v6 = deltaValue & 0x3F;
	//v5 = (_DWORD)deltaValue >> 6;
	//*(_BYTE *)lpTargetBuffer = v6;
	//if ( flag )
	//*(_BYTE *)lpTargetBuffer = v6 | 0x40;
	//
	//if ( v5 ) {
	//	v8 = lpTargetBuffer;
	//
	//	do {
	//		++idx;
	//		*(_BYTE *)v8 |= 0x80u;
	//		v8 = (char *)lpTargetBuffer + (signed __int16)idx;
	//		v9 = v5 & 0x7F;
	//		v5 >>= 7;
	//		*(_BYTE *)v8 = v9;
	//	} while ( v5 );
	//
	//	bufferLength = idx + 1;
	//
	//} else {
	//	LOWORD(bufferLength) = 1;
	//
	//}
	//
	//return bufferLength;
}
