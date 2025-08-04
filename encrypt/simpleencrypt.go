package encrypt

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
)

/**
加解密
	数据结构
	密文+sha32校验
*/
//
type SimpleEncrypt struct {
	Key string //加解密用到的key(加密key索引)+
}

// 计算检验和
func (s *SimpleEncrypt) calaCheckCode(src []byte) []byte {
	check := 0
	for i := 0; i < len(src); i++ {
		check += int(src[i])
	}
	return []byte{byte((check >> 8) & 0xff), byte(check & 0xff)}
}

// 验证数据有效性
func (s *SimpleEncrypt) verify(src []byte) bool {
	v := s.calaCheckCode(src[:len(src)-2])
	return bytes.Equal(v, src[len(src)-2:])
}

// 加密String
func (s *SimpleEncrypt) EncodeString(str string) string {
	data := []byte(str)
	s.doEncode(data)
	return base64.StdEncoding.EncodeToString(data)
}

// 加密String,ByCheck
func (s *SimpleEncrypt) EncodeStringByCheck(str string) string {
	data := []byte(str)
	s.doEncode(data)
	v := s.calaCheckCode(data)
	data = append(data, v...)
	return base64.StdEncoding.EncodeToString(data)
}

func (s *SimpleEncrypt) Encode2Hex(str string) string {
	data := []byte(str)
	s.doEncode(data)
	return hex.EncodeToString(data)
}
func (s *SimpleEncrypt) Encode2HexByCheck(str string) string {
	data := []byte(str)
	s.doEncode(data)
	v := s.calaCheckCode(data)
	data = append(data, v...)
	return hex.EncodeToString(data)
}

// 解密String
func (s *SimpleEncrypt) DecodeString(str string) string {
	data, _ := base64.StdEncoding.DecodeString(str)
	s.doEncode(data)
	return string(data)
}

// 解密String,ByCheck
func (s *SimpleEncrypt) DecodeStringByCheck(str string) string {
	data, _ := base64.StdEncoding.DecodeString(str)
	if len(data) < 2 || !s.verify(data) {
		return ""
	}
	data = data[:len(data)-2]
	s.doEncode(data)
	return string(data)
}

func (s *SimpleEncrypt) Decode4Hex(str string) string {
	data, _ := hex.DecodeString(str)
	s.doEncode(data)
	return string(data)
}
func (s *SimpleEncrypt) Decode4HexByCheck(str string) string {
	data, _ := hex.DecodeString(str)
	if len(data) < 2 || !s.verify(data) {
		return ""
	}
	data = data[:len(data)-2]
	s.doEncode(data)
	return string(data)
}

// 加密
func (s *SimpleEncrypt) Encode(data []byte) {
	s.doEncode(data)

}

func (s *SimpleEncrypt) EncodeByCheck(data []byte) {
	s.doEncode(data)
	v := s.calaCheckCode(data)
	data = append(data, v...)
}

// 解密
func (s *SimpleEncrypt) Decode(data []byte) {
	s.doEncode(data)
}

// 解密
func (s *SimpleEncrypt) DecodeByCheck(data []byte) {
	if len(data) < 2 || !s.verify(data) {
		data = []byte{}
		return
	}
	s.doEncode(data)
}

func (s *SimpleEncrypt) doEncode(bs []byte) {
	tmp := []byte(s.Key)
THEFOR:
	for i := 0; i < len(bs); {
		for j := 0; j < len(tmp); j, i = j+1, i+1 {
			if i >= len(bs) {
				break THEFOR
			}
			bs[i] = bs[i] ^ tmp[j]
		}
	}
}
