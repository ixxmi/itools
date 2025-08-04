package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var TimeFormat = "2006-01-02 15:04:05" // 默认时间戳转字符串格式
var TFormat = "2006-01-02"             // 默认时间戳转字符串格式
var TimeFormatRexMap = map[string]string{
	`^[+|-]\d{4}\s\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}$`: "-0700 2006-01-02 15:04:05",
	`^\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}$`:             "2006-01-02 15:04:05",
	`^\d{4}/\d{2}/\d{2}\s\d{2}:\d{2}:\d{2}[+|-]\d{4}$`:   "2006/01/02 15:04:05-0700",
	`^\d{4}/\d{2}/\d{2}T\d{2}:\d{2}:\d{2}[+|-]\d{4}$`:    "2006/01/02T15:04:05-0700",
	`^\d{4}/\d{2}/\d{2}\s\d{2}:\d{2}:\d{2}$`:             "2006/01/02 15:04:05",
	`^\d{4}/\d{2}/\d{2}T\d{2}:\d{2}:\d{2}$`:              "2006/01/02T15:04:05",
} // 字符串转时间戳匹配模式

var (
	// int 强制转换
	Int   = InterfaceToInt
	Int64 = InterfaceToInt64
	Int32 = InterfaceToInt32
	// float 强制转换
	Float64 = InterfaceToFloat64
	// string 强制转换
	Str = InterfaceToStr
	// bool 强制转换
	Bool = InterfaceToBool
)

// 强制转化int64
func InterfaceToInt64(x interface{}) int64 {
	switch st := reflect.ValueOf(x); st.Kind() {
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(st.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int64(st.Int())
	case reflect.Float32, reflect.Float64:
		return int64(st.Float())
	case reflect.String:
		if st.String() != "" {
			ret, err := strconv.Atoi(st.String())
			if err != nil {
				fmt.Println(err)
			}
			return int64(ret)
		}
	}

	return 0
}

// 根据指定时间格式返回时间戳
func ToTimeStamp(in string) int64 {
	var timeFormat = ""
	for r, v := range TimeFormatRexMap {
		if matched, _ := regexp.Match(r, []byte(in)); matched {
			timeFormat = v
			break
		}
	}
	if timeFormat == "" {

		return 0
	}
	ret, err := time.ParseInLocation(timeFormat, in, time.Local)
	if err != nil {
		fmt.Println(err)
	}
	return ret.Unix()
}

func FromTimeStamp(in interface{}) string {
	var data string = ""
	switch v := reflect.ValueOf(in); v.Kind() {
	case reflect.String:
		st := v.String()
		if len(st) != 0 {
			data = st
		}
		if !strings.Contains(data, ".") {
			data = data + ".0"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		data = IntToStr(int(v.Int())) + ".0"
	case reflect.Float64:
		data = strconv.FormatFloat(v.Float(), 'f', 5, 64)
	case reflect.Float32:
		data = strconv.FormatFloat(v.Float(), 'f', 5, 32)
	default:

	}
	if data == "" {
		return ""
	}
	tmpsecs := strings.Split(data, ".")
	t := time.Unix(InterfaceToInt64(tmpsecs[0]), InterfaceToInt64(tmpsecs[1])).Local()
	return t.Format(TimeFormat)

}

// json 格式返回查看数据
func Pretty(data interface{}) string {
	_data, _ := json.MarshalIndent(data, "", "    ")
	return string(_data)
}

// data 转换成ret
func Bind(data interface{}, ret interface{}) error {
	v := reflect.ValueOf(ret)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("ptr input ret needed as type as input type %s", v.Kind())
	}
	havdata := false
	var bk interface{}
	if v.Elem().Kind() == reflect.Slice {
		t := reflect.Zero(reflect.TypeOf(v.Elem().Interface()))
		bk = v.Elem().Interface()
		v.Elem().Set(t)
		havdata = true
	}
	_data, _ := json.MarshalIndent(data, "", "    ")
	err := json.Unmarshal(_data, ret)
	if err != nil {
		fmt.Println(err)
		if havdata {
			v.Elem().Set(reflect.ValueOf(bk))
		}
		return err
	}
	return nil
}

func InterfaceToInt(x interface{}) int {
	switch st := reflect.ValueOf(x); st.Kind() {
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(st.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(st.Int())
	case reflect.Float32, reflect.Float64:
		return int(st.Float())
	case reflect.String:
		if st.String() != "" {
			ret, err := strconv.Atoi(st.String())
			if err != nil {
				fmt.Println(err)
			}
			return ret
		}
	}

	return 0
}

func InterfaceToInt32(x interface{}) int32 {
	switch st := reflect.ValueOf(x); st.Kind() {
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int32(st.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int32(st.Int())
	case reflect.Float32, reflect.Float64:
		return int32(st.Float())
	case reflect.String:
		if st.String() != "" {
			ret, err := strconv.Atoi(st.String())
			if err != nil {
				fmt.Println(err)
			}
			return int32(ret)
		}
	}

	return 0
}

func InterfaceToFloat64(x interface{}) float64 {
	switch st := reflect.ValueOf(x); st.Kind() {
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(st.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(st.Int())
	case reflect.Float32, reflect.Float64:
		return float64(st.Float())
	case reflect.String:
		if st.String() != "" {
			ret, err := strconv.ParseFloat(st.String(), 64)
			if err != nil {
				fmt.Println(err)
			}
			return ret
		}
	}

	return 0
}

func InterfaceToBool(x interface{}) bool {
	var ret bool
	_ = Bind(x, &ret)
	return ret
}

func InterfaceToStr(x interface{}) string {
	switch st := reflect.ValueOf(x); st.Kind() {
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(uint64(st.Uint()), 10)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(int64(st.Int()), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(float64(st.Float()), 'g', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(st.Bool())
	}
	var ret string
	_ = Bind(x, &ret)
	return ret

}

func StrToInt(in string) int {
	if in == "" {
		return 0
	}
	rt, _ := strconv.Atoi(in)
	return rt
}

func IntToStr(in int) string {
	rt := strconv.Itoa(in)
	return rt
}

// 简单的set方法
func Set(in interface{}, out interface{}) {
	ret := map[string]interface{}{}
	ret_t := []interface{}{}
	var _in []interface{}
	_ = Bind(in, &_in)
	for _, i := range _in {
		v, _ := json.Marshal(i)
		ret[Str(v)] = i
	}
	for _, v := range ret {
		ret_t = append(ret_t, v)
	}
	_ = Bind(ret_t, out)
}

// 类型转换
func ToString(i interface{}) string {
	var ret string
	_ = Bind(i, &ret)
	return ret
}

// 将切片类型的json流byte 转换成map切片
func BJsonToListMap(bdata []byte) (jmap []map[string]interface{}) {
	err := json.Unmarshal(bdata, &jmap)
	if err != nil {
		fmt.Println(err)
		return
	}
	return
}

// 将string转换成map
func StringToMap(str string) (jmap map[string]interface{}) {
	if err := json.Unmarshal([]byte(str), &jmap); err == nil {
		return
	} else {

		return
	}
}

// 将json流 转换成map
func BJsonToMap(bdata []byte) (jmap map[string]interface{}) {
	err := json.Unmarshal(bdata, &jmap)
	if err != nil {
		fmt.Println(err)
		return
	}
	return
}

// 将切片json流字符串 转换成map切片
func SJsonToListMap(sdata string) (jmap []map[string]interface{}) {
	bdata := []byte(sdata)
	jmap = BJsonToListMap(bdata)
	return
}

// 将json流字符串 转换成map
func SJsonToMap(sdata string) (jmap map[string]interface{}) {
	bdata := []byte(sdata)
	err := json.Unmarshal(bdata, &jmap)
	if err != nil {
		fmt.Println(err)
		return
	}
	return
}

// map 转换成map字符串（jsonz字符串）
func MapToString(m interface{}) string {
	mjson, _ := json.Marshal(m)
	mString := string(mjson)
	return mString
}

// struct 转map
func StructToMap(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		data[t.Field(i).Name] = v.Field(i).Interface()
	}
	return data
}

func IterToMaps(obj interface{}) []map[string]interface{} {
	m := []map[string]interface{}{}
	j, _ := json.Marshal(obj)
	_ = json.Unmarshal(j, &m)
	return m
}

func IterToMap(obj interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	j, _ := json.Marshal(obj)
	_ = json.Unmarshal(j, &m)
	return m
}

// struct 转map
func StructToMapMore(obj interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	j, _ := json.Marshal(obj)
	_ = json.Unmarshal(j, &m)
	return m
}

// struct 转map
func StructToMapMore1(obj interface{}) map[string][]map[string]interface{} {
	m := make(map[string][]map[string]interface{})
	j, _ := json.Marshal(obj)
	_ = json.Unmarshal(j, &m)
	return m
}

// []struct to []map[string]interface{}
func ListStructToListMap(obj []interface{}) []map[string]interface{} {
	LM := []map[string]interface{}{}
	for i := 0; i < len(obj); i++ {
		m := make(map[string]interface{})
		j, _ := json.Marshal(obj[i])
		_ = json.Unmarshal(j, &m)
		LM = append(LM, m)
	}
	return LM
}

// 将map/struct 转成buffer流
func MapToBuffer(m interface{}) *bytes.Buffer {
	jsonBytes, _ := json.Marshal(m)            // 将struct转换成json串
	data := bytes.NewBuffer([]byte(jsonBytes)) // 转换成字节流切片
	return data
}

// string to int64
func StringToInt64(obj string) int64 {
	i, _ := strconv.ParseInt(obj, 10, 64)
	return i
}

// string to int
func StringToInt(obj string) int {
	i, _ := strconv.Atoi(obj)
	return i
}

// string to float64
func StringToFloat64(obj string) float64 {
	b, _ := strconv.ParseFloat(obj, 64)
	return b
}

// 将map转化成字节流
func MatToBuffer(m map[string]interface{}) *bytes.Buffer {
	jsonBytes, _ := json.Marshal(m)
	sdata := bytes.NewBuffer([]byte(jsonBytes))
	return sdata
}

// json.Number to int64  json.Number转换int64
func JsonNumberToInt64(obj json.Number) int64 {
	var b int64
	_ = json.Unmarshal([]byte(obj), &b)
	return b
}

// json.Number to float64
func JsonNumberToFloat64(obj json.Number) float64 {
	var b float64
	_ = json.Unmarshal([]byte(obj), &b)
	return b
}

// 将切片 interface转成切片字符串
func InterToSliceString(obj interface{}) []string {
	var sli = make([]string, 0)
	if obj == nil {
		return sli
	}

	if _, ok := obj.([]interface{}); ok {
		for _, i := range obj.([]interface{}) {
			sli = append(sli, i.(string))
		}
	}

	if _, ok := obj.([]string); ok {
		return obj.([]string)
	}

	return sli
}

// 将切片 interface转成切片字符串
func SliceInterToString(obj []interface{}) (sli []string) {
	for _, i := range obj {
		sli = append(sli, i.(string))
	}
	return sli
}

// 将切片 interface转成切片字符串
func Float64ToStr(obj float64) (sli string) {
	str := strconv.FormatFloat(obj, 'G', -1, 64)
	return str
}

// interface to map interface
func ToSlice(arr interface{}) []interface{} {
	v := reflect.ValueOf(arr)
	if v.Kind() != reflect.Slice {
		return []interface{}{}
	}
	l := v.Len()
	ret := make([]interface{}, l)
	for i := 0; i < l; i++ {
		ret[i] = v.Index(i).Interface()
	}
	return ret
}

// []interface to []map[string]interface{}
func InToMap(b []interface{}) []map[string]interface{} {
	var d []map[string]interface{}
	for n := range b {
		value, ok := b[n].(map[string]interface{})
		if !ok {
			return []map[string]interface{}{}
		} else {
			d = append(d, value)
		}
	}
	return d
}

func ToInt(x interface{}) int {
	switch st := reflect.ValueOf(x); st.Kind() {
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(st.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(st.Int())
	case reflect.Float32, reflect.Float64:
		return int(st.Float())
	case reflect.String:
		if st.String() != "" {
			ret, err := strconv.Atoi(st.String())
			if err != nil {
				fmt.Println(err)
			}
			return ret
		}
	}

	return 0
}

// 排序 排序键必须为数字类型
type SortBy struct {
	Data    []map[string]interface{}
	Sortkey string
}

func (a SortBy) Len() int { return len(a.Data) }

func (a SortBy) Swap(i, j int) {
	a.Data[i], a.Data[j] = a.Data[j], a.Data[i]
}

func (a SortBy) Less(i, j int) bool {
	//return Float64(a.Data[i][a.Sortkey]) < Float64(a.Data[j][a.Sortkey])
	m := a.Data[i][a.Sortkey]
	n := a.Data[j][a.Sortkey]
	w := reflect.ValueOf(m)
	v := reflect.ValueOf(n)
	switch v.Kind() {
	case reflect.String:
		return w.String() < v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return w.Int() < v.Int()
	case reflect.Float64, reflect.Float32:
		return w.Float() < v.Float()
	default:
		return fmt.Sprintf("%v", w) < fmt.Sprintf("%v", v)
	}
}

// 根据指定字符排序
//
//	m := []map[string]int{
//		{"k": 2},
//		{"k": 1},
//		{"k": 3},
//	}
//
// customer.SortData(&m, "k", true)
// ture  倒序3, 2, 1
// fmt.Println(m)
func SortData(data interface{}, sortkey string, reverse bool) {
	//func SortData(data interface{}, sortkey string, reverse bool) {
	var db []map[string]interface{}
	err := Bind(data, &db)
	if err != nil {
		fmt.Println(err)
		return
	}
	stb := SortBy{db, sortkey}
	if !reverse {
		sort.Sort(stb)
	} else {
		sort.Sort(sort.Reverse(stb))
	}
	err = Bind(stb.Data, data)
	if err != nil {
		fmt.Println(err)
	}

}

const ( // 格式化基本时间范式 固定时间不可更改
	LicenseLoginFileName = "/etc/service/license/license_login.conf" // 正式版本路径
	UptimeFile           = "/proc/uptime"                            //linux
	//////////
	POST    = "POST"
	GET     = "GET"
	HEAD    = "HEAD"
	PUT     = "PUT"
	DELETE  = "DELETE"
	PATCH   = "PATCH"
	OPTIONS = "OPTIONS"
)

// 获取公共平台访问地址、端口类型设定
type Cloud_env_Get struct {
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

// 全局通用 返回请求数据strcut
type SendData struct {
	Success bool        `json:"success" bson:"success"`
	Ret     interface{} `json:"ret" bson:"ret"`
}

// 全局
type SaveDebug struct {
	UserID   string      `json:"user_id" bson:"user_id" form:"user_id" query:"user_id"  validate:"user_id"`
	UserRole string      `json:"user_role" bson:"user_role" form:"user_role" query:"user_role"  validate:"user_role"`
	ClientIp string      `json:"client_ip" bson:"client_ip" form:"client_ip" query:"client_ip"  validate:"client_ip"`
	Optype   string      `json:"optype" bson:"optype" form:"optype" query:"optype"  validate:"optype"`
	Content  interface{} `json:"content" bson:"content" form:"content" query:"content"  validate:"content"`
	Ret      interface{} `json:"ret" bson:"ret" form:"ret" query:"ret"  validate:"ret"`
	Time     int         `json:"time" bson:"time" form:"time" query:"time"  validate:"time"`
	ErrorMsg string      `json:"error_msg" bson:"error_msg" form:"error_msg" query:"error_msg"  validate:"error_msg"`
}

// open and read
func FileOpenRead(filename string) (contents []uint8, err error) {

	if fileObj, err := os.OpenFile(filename, os.O_RDWR, 0755); err == nil {
		defer fileObj.Close()
		if contents, err := ioutil.ReadAll(fileObj); err == nil {
			return contents, err
		} else {
			e := fmt.Sprintf("fileReadAllFlase: %v file open or read  error -->%v", filename, err)
			err := errors.New(e)
			return contents, err
		}
	} else {
		e := fmt.Sprintf("openFileFalse: %v file open  error -->%v", filename, err)
		err := errors.New(e)
		return contents, err
	}
	//return contents, err
}

// readlines  读取全部行 返回列表
func Readlines(filename string) (readlineslist []string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		e := fmt.Sprintf("file open falied %s", err)
		err := errors.New(e)
		return nil, err
	}
	defer f.Close()
	rf := bufio.NewReader(f)
	for {
		b, err := rf.ReadBytes('\n')
		readlineslist = append(readlineslist, string(b))
		if err == io.EOF {
			break
		}
	}
	return readlineslist, nil
}

// 判断文件是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// 类型转换
func toString(i interface{}) string {
	var ret string
	_ = Bind(i, &ret)
	return ret
}

// struct 转map
func structToMap(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		data[t.Field(i).Name] = v.Field(i).Interface()
	}
	return data
}

// 向上取整
func MathCeil(x float64) int {
	return int(math.Ceil(x + 0/5))
}

// 向下取整
func MathFloor(x float64) int {
	return int(math.Floor(x + 0/5))
}

// 对切片map排序  冒泡排序  mapslice 需要排序的map类型的切片 ，sortkey,排序key键关键字 ， direction true为递增排序false为递减排序
func SortedMap(mapslice []map[string]interface{}, sortkey string, direction bool) []map[string]interface{} {
	for i := 0; i < len(mapslice)-1; i++ {
		for j := i + 1; j < len(mapslice); j++ {

			//加一个提醒
			if _, ok := mapslice[i][sortkey]; !ok {
				fmt.Printf("error key %s is not exist Please check map \n", sortkey)
			}

			if direction {
				switch mapslice[i][sortkey].(type) {
				case string:
					if mapslice[i][sortkey].(string) > mapslice[j][sortkey].(string) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				case int:
					if mapslice[i][sortkey].(int) > mapslice[j][sortkey].(int) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				case int32:
					if mapslice[i][sortkey].(int32) > mapslice[j][sortkey].(int32) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				case int64:
					if mapslice[i][sortkey].(int64) > mapslice[j][sortkey].(int64) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				case float64:
					if mapslice[i][sortkey].(float64) > mapslice[j][sortkey].(float64) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				}

			} else {
				switch mapslice[i][sortkey].(type) {
				case string:
					if mapslice[i][sortkey].(string) < mapslice[j][sortkey].(string) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				case int:
					if mapslice[i][sortkey].(int) < mapslice[j][sortkey].(int) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				case int32:
					if mapslice[i][sortkey].(int32) < mapslice[j][sortkey].(int32) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				case int64:
					if mapslice[i][sortkey].(int64) < mapslice[j][sortkey].(int64) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				case float64:
					if mapslice[i][sortkey].(float64) < mapslice[j][sortkey].(float64) {
						mapslice[i], mapslice[j] = mapslice[j], mapslice[i]
					}
				}
			}
		}
	}
	return mapslice
}

// 遍历轮询切片
func Rangelist(b interface{}, list []interface{}) bool {
	if len(list) == 0 {
		return false
	}
	for _, i := range list {
		if i == b {
			return true
		}
	}
	return false
}

// 判断是否为闰年
func IsLeapYear(year int) bool { //y == 2000, 2004
	//判断是否为闰年
	if year%4 == 0 && year%100 != 0 || year%400 == 0 {
		return true
	}
	return false
}

// 调用系统命令返回内容 返回运行系统时区
func Formattimezone() string {
	now := time.Now()
	local2, err := time.LoadLocation("Local") //服务器设置的时区
	if err != nil {
		fmt.Println(err)
		return ""
	}
	shiqu, _ := now.In(local2).Zone()
	return shiqu
}

// 对两个切片字符串去重并合并
func SetListStringadd(list1, list2 []string) (listAdd []string) {
	listAdd = RemoveRepeatedElement(append(list1, list2...))
	return
}

// 对两个切片字符串去重并相减
func SetListString(list1, list2 []string) []string {
	obj := RemoveRepeatedElement(list1)
	list1 = RemoveRepeatedElement(list1)
	list2 = RemoveRepeatedElement(list2)
	for i := 0; i < len(list1); i++ {
		for j := 0; j < len(list2); j++ {
			if list2[j] == list1[i] {
				if i < len(obj) {
					obj = append(obj[:i], obj[i+1:]...)
				} else {
					obj = obj[:len(obj)-1]
				}
				break
			}
		}
	}
	return obj
}

// 去重列表内容
func RemoveRepeatedElement(arr []string) (newArr []string) {
	newArr = make([]string, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

// 去重列表内容
func RemoveRepeatedElementList(arr []interface{}) (newArr []interface{}) {
	newArr = make([]interface{}, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

// range  []map[string]interface{}判断map是否存在一个列表中
func RangeListMap(obj map[string]interface{}, list []map[string]interface{}) bool {
	if len(list) == 0 {
		return false
	}
	for _, temp := range list {
		for k, v := range temp {
			if obj[k] == v {
				return true
			}
		}
	}
	return false
}

// 判断map切片是否存在一个key
func RangeKeyListMap(obj interface{}, list []map[string]interface{}) bool {
	if len(list) == 0 {
		return false
	}
	for _, temp := range list {
		if temp[obj.(string)] != nil {
			return true
		}
	}
	return false
}

// 判断map是否存在一个key
func RangeMap(obj interface{}, list map[string]interface{}) bool {
	if len(list) == 0 {
		return false
	}
	for key := range list {
		if key == obj.(string) {
			return true
		}
	}
	return false
}

// map数组转成对象数组
func MapArrToObjArr(old []map[string]interface{}) []interface{} {
	if old != nil {
		new := make([]interface{}, len(old))
		for i, v := range old {
			new[i] = v
		}
		return new
	} else {
		return nil
	}
}

// 3目运算
func If(b bool, to, fo interface{}) interface{} {
	if b {
		return to
	} else {
		return fo
	}
}

func StartWith(value, str string) bool {
	ok, _ := regexp.MatchString("^"+str, value)
	return ok
}
func EndWith(value, str string) bool {
	ok, _ := regexp.MatchString(str+"$", value)
	return ok
}

// 数组去重
func RemoveDuplicatesAndEmpty(a []string) (ret []string) {
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1] == a[i]) || len(a[i]) == 0 {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}

// 验证邮箱
func VerifyEmailFormat(email string) bool {
	pattern := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*` //匹配电子邮箱
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(email)
}

// map转str
func MapToStr(param map[string]interface{}) string {
	dataType, _ := json.Marshal(param)
	dataString := string(dataType)
	return dataString
}

// str转map
func StrToMap(str string) map[string]interface{} {

	var tempMap map[string]interface{}

	err := json.Unmarshal([]byte(str), &tempMap)

	if err != nil {
		panic(err)
	}

	return tempMap
}

// 排序(slow)
type SortByEx struct {
	Data    []map[string]interface{}
	Sortkey []string
}

func (a SortByEx) Len() int { return len(a.Data) }

func (a SortByEx) Swap(i, j int) {
	a.Data[i], a.Data[j] = a.Data[j], a.Data[i]
}

func (a SortByEx) LessSub(keyindex, i, j int) bool {
	if keyindex > 20 {
		fmt.Println("step is to deep as", keyindex)
		return true
	}
	if keyindex > len(a.Sortkey)-1 {
		fmt.Println(keyindex, "index skip")
		return true
	}
	m := a.Data[i][a.Sortkey[keyindex]]
	n := a.Data[j][a.Sortkey[keyindex]]
	w := reflect.ValueOf(m)
	v := reflect.ValueOf(n)
	switch v.Kind() {
	case reflect.String:
		if w.String() == v.String() {
			return a.LessSub(keyindex+1, i, j)
		}
		return w.String() < v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if w.Int() == v.Int() {
			return a.LessSub(keyindex+1, i, j)
		}
		return w.Int() < v.Int()
	case reflect.Float64, reflect.Float32:
		if w.Float() == v.Float() {
			return a.LessSub(keyindex+1, i, j)
		}
		return w.Float() < v.Float()
	default:
		if fmt.Sprintf("%v", w) == fmt.Sprintf("%v", v) {
			return a.LessSub(keyindex+1, i, j)
		}
		return fmt.Sprintf("%v", w) < fmt.Sprintf("%v", v)
	}
}

func (a SortByEx) Less(i, j int) bool {
	//return Float64(a.Data[i][a.Sortkey]) < Float64(a.Data[j][a.Sortkey])
	m := a.Data[i][a.Sortkey[0]]
	n := a.Data[j][a.Sortkey[0]]
	w := reflect.ValueOf(m)
	v := reflect.ValueOf(n)
	switch v.Kind() {
	case reflect.String:
		if w.String() == v.String() {
			return a.LessSub(1, i, j)
		}
		return w.String() < v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if w.Int() == v.Int() {
			return a.LessSub(1, i, j)
		}
		return w.Int() < v.Int()
	case reflect.Float64, reflect.Float32:
		if w.Float() == v.Float() {
			return a.LessSub(1, i, j)
		}
		return w.Float() < v.Float()
	default:
		if fmt.Sprintf("%v", w) == fmt.Sprintf("%v", v) {
			return a.LessSub(1, i, j)
		}
		return fmt.Sprintf("%v", w) < fmt.Sprintf("%v", v)
	}
}
func SortDataEx(data interface{}, sortkey []string, reverse bool) {
	var db []map[string]interface{}
	err := Bind(data, &db)
	if err != nil {
		fmt.Println(3, "sortdata error", err)
		return
	}
	stb := SortByEx{db, sortkey}
	if !reverse {
		sort.Sort(stb)
	} else {
		sort.Sort(sort.Reverse(stb))
	}
	err = Bind(stb.Data, data)
	if err != nil {
		fmt.Println(3, "sortdata error", err)
	}
}

// 获取昨天0点起止时间
func GetYesterdayTime() []string {
	var yTime []string
	ts := time.Now().AddDate(0, 0, -1)
	timeYesterDay := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location()).Unix()
	timeStr := time.Unix(timeYesterDay, 0).Format("2006-01-02")
	yTime = append(yTime, timeStr+" 00:00:00", timeStr+" 23:59:59")
	return yTime
}
func JoinRedisKey(key ...string) string {
	for i, _ := range key {
		if key[i] == "" {
			key[i] = ""
		}
	}
	str := strings.Join(key, ":")
	return str
}

// 当前小时上一个小时时间段计算
func LastDuration() (sTime, eTime string) {
	lastHour := time.Now().Add(-1 * 60 * time.Minute).Format("2006-01-02 15:04:05")
	listT := strings.Split(lastHour, " ")
	year := listT[0]
	lTime := strings.Split(listT[1], ":")[0]
	bTime := fmt.Sprintf(" %s:00:00", lTime)
	etime := fmt.Sprintf(" %s:59:59", lTime)
	sTime = year + bTime
	eTime = year + etime
	return
}

// 获取一个数组里最大值，并且拿到下标
func ListMaxValInt(slice interface{}) (maxIndex, minIndex int, maxVal, minVal, avgVal float64) {
	arr := InterToSliceString(slice)
	//假设第一个元素是最大值，下标为0
	maxVal = InterfaceToFloat64(arr[0])
	minVal = InterfaceToFloat64(arr[0])
	maxIndex = 0
	minIndex = 0
	var arrLen int

	for i := 1; i < len(arr); i++ {
		if maxVal == float64(65535) {
			maxVal = InterfaceToFloat64(arr[i])
			continue
		}
		if InterfaceToFloat64(arr[i]) == float64(65535) {
			continue
		}
		arrLen += 1
		avgVal += InterfaceToFloat64(arr[i])
		//从第二个 元素开始循环比较，如果发现有更大的，则交换
		if maxVal < InterfaceToFloat64(arr[i]) {
			maxVal = InterfaceToFloat64(arr[i])
			maxIndex = i
		}
		if minVal > InterfaceToFloat64(arr[i]) {
			minVal = InterfaceToFloat64(arr[i])
			minIndex = i
		}
	}
	avgVal = avgVal / float64(arrLen)
	return
}

func ListMaxVal(list []int) (maxval int) {
	maxval = list[0]
	for i := 1; i < len(list); i++ {
		if maxval < list[i] {
			maxval = list[i]
		}
	}
	return
}

// 根据时间戳获取时间
func UnixToTime(t, typ int64) (sTime string) {
	if typ == 0 {
		sTime = time.Unix(time.Now().Unix()-t, 0).Format("2006-01-02 15:04:05")
	} else {
		sTime = time.Unix(time.Now().Unix()+t, 0).Format("2006-01-02 15:04:05")
	}
	return
}

func StringToSlice(s string) []string {
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")
	return strings.Split(s, ",")
}

func SliceToStr(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func CheckIp(ipAddr string) bool {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		fmt.Println("Invalid IP address")
		return false
	}
	return true
}

// ReadFile 读取文件内容
func ReadFile(filename string) (string, error) {
	// 使用 ioutil.ReadFile 读取文件内容
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %v", filename, err)
	}
	return string(data), nil
}

// WriteFile 将内容写入文件
func WriteFile(filename string, data []byte) error {
	// 使用 ioutil.WriteFile 写入文件内容
	err := ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %v", filename, err)
	}
	return nil
}
