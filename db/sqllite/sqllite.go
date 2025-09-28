package sl

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// DBHelper 通用数据库操作辅助类
type DBHelper struct {
	db *sql.DB
}

// NewDBHelper 创建新的数据库辅助实例
func NewDBHelper(dbPath string) (*DBHelper, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %v", err)
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %v", err)
	}

	return &DBHelper{db: db}, nil
}

// CreateTableFromStruct 根据结构体自动创建表.
// 使用 GORM-style 的 db 标签来定义字段属性.
// 标签格式: `db:"column_name;option1;option2:value;..."`
// 支持的选项:
// - primaryKey: 主键
// - autoIncrement: 自增
// - not null: 非空
// - unique: 唯一
// - index: 创建索引
// - size:255: 字段大小 (用于 VARCHAR)
// - default:'some_value': 默认值
func (h *DBHelper) CreateTableFromStruct(tableName string, structType interface{}) error {
	t := reflect.TypeOf(structType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var columns []string
	var indexes []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")

		if dbTag == "" || dbTag == "-" {
			continue
		}

		// GORM-style tag parsing
		parts := strings.Split(dbTag, ";")
		if len(parts) == 0 {
			continue
		}

		// The first part is always the column name
		columnName := strings.TrimSpace(parts[0])
		if columnName == "" {
			continue
		}

		// Get base SQL type from Go type
		sqlType := h.getSQLType(field.Type)

		// Process other tags
		var colDefs []string

		for _, part := range parts[1:] {
			option := strings.TrimSpace(part)
			if option == "" {
				continue
			}

			kv := strings.SplitN(option, ":", 2)
			key := strings.ToLower(kv[0])
			value := ""
			if len(kv) > 1 {
				value = kv[1]
			}

			switch key {
			case "primarykey", "primary_key":
				colDefs = append(colDefs, "PRIMARY KEY")
			case "autoincrement", "auto_increment":
				colDefs = append(colDefs, "AUTOINCREMENT")
			case "not null":
				colDefs = append(colDefs, "NOT NULL")
			case "unique":
				colDefs = append(colDefs, "UNIQUE")
			case "index":
				indexes = append(indexes, columnName)
			case "size":
				if value != "" && (sqlType == "TEXT" || sqlType == "VARCHAR") {
					sqlType = fmt.Sprintf("VARCHAR(%s)", value)
				}
			case "default":
				colDefs = append(colDefs, fmt.Sprintf("DEFAULT %s", value))
			}
		}

		// Assemble the column definition string
		finalColumnDef := fmt.Sprintf("%s %s %s", columnName, sqlType, strings.Join(colDefs, " "))
		columns = append(columns, strings.TrimSpace(finalColumnDef))
	}

	// Create table SQL
	createTableSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		tableName, strings.Join(columns, ",\n  "))

	_, err := h.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}

	// Create indexes
	for _, indexColumn := range indexes {
		indexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s(%s)",
			tableName, indexColumn, tableName, indexColumn)
		_, err := h.db.Exec(indexSQL)
		if err != nil {
			return fmt.Errorf("创建索引失败: %v", err)
		}
	}

	return nil
}

// getSQLType 根据Go类型获取对应的SQL类型
func (h *DBHelper) getSQLType(goType reflect.Type) string {
	switch goType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "INTEGER"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.String:
		return "TEXT"
	case reflect.Slice:
		if goType.Elem().Kind() == reflect.Uint8 {
			return "BLOB"
		}
		return "TEXT"
	case reflect.Array:
		if goType.Elem().Kind() == reflect.Uint8 {
			return "BLOB"
		}
		return "TEXT"
	case reflect.Map, reflect.Struct:
		// time.Time 是一个特殊的结构体，需要有自己的DATETIME类型
		if goType == reflect.TypeOf(time.Time{}) {
			return "DATETIME"
		}
		return "TEXT"
	}

	// 其他未处理的类型默认为TEXT
	return "TEXT"
}

// Insert 插入记录
func (h *DBHelper) Insert(tableName string, data interface{}) error {
	v := reflect.ValueOf(data)
	t := reflect.TypeOf(data)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	var columns []string
	var placeholders []string
	var values []interface{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")

		if dbTag == "" || dbTag == "-" {
			continue
		}

		parts := strings.Split(dbTag, ";")
		columnName := strings.TrimSpace(parts[0])
		if columnName == "" {
			continue
		}

		// 跳过自增字段
		if h.hasOption(parts, "autoIncrement") || h.hasOption(parts, "autoincrement") {
			continue
		}

		fieldValue := v.Field(i)
		columns = append(columns, columnName)
		placeholders = append(placeholders, "?")

		// 处理不同类型的值
		processedValue, err := h.processFieldValue(fieldValue)
		if err != nil {
			return fmt.Errorf("处理字段 %s 失败: %v", columnName, err)
		}

		values = append(values, processedValue)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	_, err := h.db.Exec(query, values...)
	return err
}

// BatchInsert 批量插入记录
func (h *DBHelper) BatchInsert(tableName string, data interface{}) error {
	slice := reflect.ValueOf(data)
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("data必须是切片类型")
	}

	if slice.Len() == 0 {
		return nil // 没有数据需要插入
	}

	// 从第一个元素获取类型信息来构建SQL语句
	first := slice.Index(0)
	t := first.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var columns []string
	var placeholders []string
	var fieldIndexes []int // 存储需要插入的字段索引

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")

		if dbTag == "" || dbTag == "-" {
			continue
		}

		parts := strings.Split(dbTag, ";")
		columnName := strings.TrimSpace(parts[0])
		if columnName == "" {
			continue
		}

		// 跳过自增字段
		if h.hasOption(parts, "autoIncrement") || h.hasOption(parts, "autoincrement") {
			continue
		}

		columns = append(columns, columnName)
		placeholders = append(placeholders, "?")
		fieldIndexes = append(fieldIndexes, i)
	}

	if len(columns) == 0 {
		return fmt.Errorf("没有可插入的字段")
	}

	// 开始事务
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	// 使用defer确保在出错时回滚
	defer tx.Rollback()

	// 准备SQL语句
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("预编译SQL失败: %v", err)
	}
	defer stmt.Close()

	// 遍历切片并执行插入
	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		var values []interface{}
		for _, fieldIndex := range fieldIndexes {
			fieldValue := item.Field(fieldIndex)
			processedValue, err := h.processFieldValue(fieldValue)
			if err != nil {
				return fmt.Errorf("处理第 %d 条记录的字段 %s 失败: %v", i, t.Field(fieldIndex).Name, err)
			}
			values = append(values, processedValue)
		}

		_, err := stmt.Exec(values...)
		if err != nil {
			return fmt.Errorf("执行第 %d 条记录插入失败: %v", i, err)
		}
	}

	// 提交事务
	return tx.Commit()
}

// processFieldValue 处理字段值，特殊处理slice类型
func (h *DBHelper) processFieldValue(fieldValue reflect.Value) (interface{}, error) {
	switch fieldValue.Kind() {
	case reflect.Slice:
		// 处理字节切片
		if fieldValue.Type().Elem().Kind() == reflect.Uint8 {
			return fieldValue.Interface(), nil
		}

		// 处理其他类型的切片，转换为JSON字符串
		sliceData := fieldValue.Interface()
		jsonData, err := json.Marshal(sliceData)
		if err != nil {
			return nil, fmt.Errorf("slice转换JSON失败: %v", err)
		}
		return string(jsonData), nil

	case reflect.Array:
		// 为保持与[]byte一致，将[N]byte作为BLOB处理
		if fieldValue.Type().Elem().Kind() == reflect.Uint8 {
			// 必须将[N]byte转换为[]byte，数据库驱动才能识别
			if fieldValue.CanAddr() {
				return fieldValue.Slice(0, fieldValue.Len()).Interface(), nil
			}
			// 如果不可寻址，则复制
			slice := reflect.MakeSlice(reflect.TypeOf([]byte{}), fieldValue.Len(), fieldValue.Len())
			reflect.Copy(slice, fieldValue)
			return slice.Interface(), nil
		}

		// 处理其他类型的数组，转换为JSON字符串
		arrayData := fieldValue.Interface()
		jsonData, err := json.Marshal(arrayData)
		if err != nil {
			return nil, fmt.Errorf("array转换JSON失败: %v", err)
		}
		return string(jsonData), nil

	case reflect.Map:
		// 处理map，转换为JSON字符串
		mapData := fieldValue.Interface()
		jsonData, err := json.Marshal(mapData)
		if err != nil {
			return nil, fmt.Errorf("map转换JSON失败: %v", err)
		}
		return string(jsonData), nil

	case reflect.Struct:
		// 处理结构体（除了time.Time）
		if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
			return fieldValue.Interface(), nil
		}

		// 其他结构体转换为JSON字符串
		structData := fieldValue.Interface()
		jsonData, err := json.Marshal(structData)
		if err != nil {
			return nil, fmt.Errorf("struct转换JSON失败: %v", err)
		}
		return string(jsonData), nil

	case reflect.Ptr:
		// 处理指针
		if fieldValue.IsNil() {
			return nil, nil
		}
		return h.processFieldValue(fieldValue.Elem())

	default:
		// 基本类型直接返回
		return fieldValue.Interface(), nil
	}
}

// QueryToMaps 执行查询并返回一个 map 切片 ([]map[string]interface{})
func (h *DBHelper) QueryMaps(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %v", err)
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %v", err)
		}

		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]

			// 如果值是[]byte类型，则转换为string
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("处理行时出错: %v", err)
	}

	return results, nil
}

// QueryOneToMap 查询单行记录并将其作为 map[string]interface{} 返回
func (h *DBHelper) QueryMap(query string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %v", err)
	}

	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	if err := rows.Scan(scanArgs...); err != nil {
		return nil, fmt.Errorf("扫描行数据失败: %v", err)
	}

	rowMap := make(map[string]interface{})
	for i, colName := range columns {
		val := values[i]
		if b, ok := val.([]byte); ok {
			rowMap[colName] = string(b)
		} else {
			rowMap[colName] = val
		}
	}

	return rowMap, rows.Err()
}

// Query 通用查询方法
func (h *DBHelper) Query(query string, result interface{}, args ...interface{}) error {
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr {
		return fmt.Errorf("result必须是指针类型")
	}

	sliceValue := resultValue.Elem()
	if sliceValue.Kind() != reflect.Slice {
		return fmt.Errorf("result必须是切片指针")
	}

	elementType := sliceValue.Type().Elem()
	if elementType.Kind() == reflect.Ptr {
		elementType = elementType.Elem()
	}

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		// 创建新的结构体实例
		newElem := reflect.New(elementType).Elem()

		// 创建扫描目标
		scanArgs := make([]interface{}, len(columns))
		fieldMap := make(map[string]reflect.Value)

		for i, columnName := range columns {
			field := h.findFieldByDBTag(elementType, columnName)
			if field != nil {
				fieldValue := newElem.FieldByName(field.Name)
				fieldMap[columnName] = fieldValue

				// 为复杂类型创建临时字符串存储
				if h.isComplexType(field.Type) {
					var tempStr sql.NullString
					scanArgs[i] = &tempStr
				} else {
					scanArgs[i] = fieldValue.Addr().Interface()
				}
			} else {
				var dummy interface{}
				scanArgs[i] = &dummy
			}
		}

		// 扫描数据
		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}

		// 处理复杂类型的反序列化
		for i, columnName := range columns {
			field := h.findFieldByDBTag(elementType, columnName)
			if field != nil && h.isComplexType(field.Type) {
				fieldValue := fieldMap[columnName]
				tempStr := scanArgs[i].(*sql.NullString)

				if tempStr.Valid && tempStr.String != "" {
					if err := h.deserializeFieldValue(fieldValue, tempStr.String); err != nil {
						return fmt.Errorf("反序列化字段 %s 失败: %v", columnName, err)
					}
				}
			}
		}

		// 添加到结果切片
		if sliceValue.Type().Elem().Kind() == reflect.Ptr {
			sliceValue.Set(reflect.Append(sliceValue, newElem.Addr()))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, newElem))
		}
	}

	return nil
}

// QueryRaw 执行原始SQL查询并返回 []map[string]interface{}
func (h *DBHelper) QueryRaw(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %v", err)
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %v", err)
		}

		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]

			// 如果值是[]byte类型，则转换为string
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("处理行时出错: %v", err)
	}

	return results, nil
}

// QueryOne 查询单个记录
func (h *DBHelper) QueryOne(query string, result interface{}, args ...interface{}) error {
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.IsNil() {
		return fmt.Errorf("result必须是非空的指针类型")
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	elemValue := resultValue.Elem()

	// 如果目标是结构体，使用现有的复杂扫描逻辑
	if elemValue.Kind() == reflect.Struct {
		elemType := elemValue.Type()
		columns, err := rows.Columns()
		if err != nil {
			return err
		}

		scanArgs := make([]interface{}, len(columns))
		fieldMap := make(map[string]reflect.Value)

		for i, columnName := range columns {
			field := h.findFieldByDBTag(elemType, columnName)
			if field != nil {
				fieldValue := elemValue.FieldByName(field.Name)
				fieldMap[columnName] = fieldValue

				if h.isComplexType(field.Type) {
					var tempStr sql.NullString
					scanArgs[i] = &tempStr
				} else {
					scanArgs[i] = fieldValue.Addr().Interface()
				}
			} else {
				var dummy interface{}
				scanArgs[i] = &dummy
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}

		for i, columnName := range columns {
			field := h.findFieldByDBTag(elemType, columnName)
			if field != nil && h.isComplexType(field.Type) {
				fieldValue := fieldMap[columnName]
				tempStr := scanArgs[i].(*sql.NullString)

				if tempStr.Valid && tempStr.String != "" {
					if err := h.deserializeFieldValue(fieldValue, tempStr.String); err != nil {
						return fmt.Errorf("反序列化字段 %s 失败: %v", columnName, err)
					}
				}
			}
		}
	} else {
		// 如果目标是基本类型，则直接扫描
		err = rows.Scan(result)
		if err != nil {
			return err
		}
	}

	return rows.Err()
}

// deserializeFieldValue 反序列化字段值
func (h *DBHelper) deserializeFieldValue(fieldValue reflect.Value, jsonStr string) error {
	// 创建目标类型的新实例
	targetType := fieldValue.Type()
	newValue := reflect.New(targetType)

	// 反序列化JSON
	if err := json.Unmarshal([]byte(jsonStr), newValue.Interface()); err != nil {
		return fmt.Errorf("JSON反序列化失败: %v", err)
	}

	// 设置字段值
	fieldValue.Set(newValue.Elem())
	return nil
}

// isComplexType 判断是否为复杂类型（需要JSON序列化的类型）
func (h *DBHelper) isComplexType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Slice:
		// 字节切片除外
		return t.Elem().Kind() != reflect.Uint8
	case reflect.Array:
		// 字节数组除外
		return t.Elem().Kind() != reflect.Uint8
	case reflect.Map:
		return true
	case reflect.Struct:
		// time.Time除外
		return t != reflect.TypeOf(time.Time{})
	case reflect.Ptr:
		// 指向复杂类型的指针
		return h.isComplexType(t.Elem())
	default:
		return false
	}
}

// findFieldByDBTag 根据db标签查找字段
func (h *DBHelper) findFieldByDBTag(structType reflect.Type, dbTag string) *reflect.StructField {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ";")
		columnNameInTag := strings.TrimSpace(parts[0])
		if columnNameInTag == dbTag {
			return &field
		}
	}
	return nil
}

// hasOption 检查标签是否包含特定选项
func (h *DBHelper) hasOption(parts []string, option string) bool {
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == option {
			return true
		}
	}
	return false
}

// Close 关闭数据库连接
func (h *DBHelper) Close() error {
	return h.db.Close()
}

// CleanupRule 清理规则
type CleanupRule struct {
	TableName       string        `json:"table_name"`
	TimestampColumn string        `json:"timestamp_column"`
	RetentionPeriod time.Duration `json:"retention_period"`
	Enabled         bool          `json:"enabled"`
}

// AlgorithmLogger 算法数据记录器
type AlgorithmLogger struct {
	DbHelper     *DBHelper
	CleanupRules []CleanupRule
}

// NewAlgorithmLogger 创建新的算法记录器
func NewAlgorithmLogger(dbPath string) (*AlgorithmLogger, error) {
	helper, err := NewDBHelper(dbPath)
	if err != nil {
		return nil, err
	}

	logger := &AlgorithmLogger{
		DbHelper:     helper,
		CleanupRules: []CleanupRule{},
	}

	// 启动定时清理任务
	go logger.startCleanupTask()

	return logger, nil
}

// Statistics 统计信息
type Statistics struct {
	TotalCount  int64   `db:"total_count"`
	AvgDuration float64 `db:"avg_duration"`
	MinDuration int64   `db:"min_duration"`
	MaxDuration int64   `db:"max_duration"`
}

// GetStatistics 获取统计信息
func (al *AlgorithmLogger) GetStatistics(start, end time.Time) (*Statistics, error) {
	var stats Statistics

	query := `SELECT 
				COUNT(*) as total_count,
				AVG(duration) as avg_duration,
				MIN(duration) as min_duration,
				MAX(duration) as max_duration
			  FROM algorithm_records
			  WHERE timestamp BETWEEN ? AND ?`

	err := al.DbHelper.QueryOne(query, &stats, start, end)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// AddCleanupRule 添加清理规则
func (al *AlgorithmLogger) AddCleanupRule(tableName, timestampColumn string, retentionPeriod time.Duration) {
	rule := CleanupRule{
		TableName:       tableName,
		TimestampColumn: timestampColumn,
		RetentionPeriod: retentionPeriod,
		Enabled:         true,
	}

	// 检查是否已存在该表的规则
	for i, existingRule := range al.CleanupRules {
		if existingRule.TableName == tableName {
			al.CleanupRules[i] = rule
			return
		}
	}

	// 添加新规则
	al.CleanupRules = append(al.CleanupRules, rule)
}

// RemoveCleanupRule 移除清理规则
func (al *AlgorithmLogger) RemoveCleanupRule(tableName string) {
	for i, rule := range al.CleanupRules {
		if rule.TableName == tableName {
			al.CleanupRules = append(al.CleanupRules[:i], al.CleanupRules[i+1:]...)
			return
		}
	}
}

// SetCleanupRuleEnabled 启用/禁用清理规则
func (al *AlgorithmLogger) SetCleanupRuleEnabled(tableName string, enabled bool) {
	for i, rule := range al.CleanupRules {
		if rule.TableName == tableName {
			al.CleanupRules[i].Enabled = enabled
			return
		}
	}
}

// GetCleanupRules 获取所有清理规则
func (al *AlgorithmLogger) GetCleanupRules() []CleanupRule {
	return al.CleanupRules
}

// cleanupOldRecords 根据配置的规则清理旧记录
func (al *AlgorithmLogger) CleanupOldRecords() error {
	totalDeleted := int64(0)

	// 如果没有配置规则，使用自动发现模式
	if len(al.CleanupRules) == 0 {
		return al.cleanupOldRecordsAuto()
	}

	// 根据配置的规则进行清理
	for _, rule := range al.CleanupRules {
		if !rule.Enabled {
			continue
		}

		cutoff := time.Now().Add(-rule.RetentionPeriod)

		// 检查表是否存在
		if !al.tableExists(rule.TableName) {
			log.Printf("表 %s 不存在，跳过清理", rule.TableName)
			continue
		}

		// 检查时间戳字段是否存在
		if !al.columnExists(rule.TableName, rule.TimestampColumn) {
			log.Printf("表 %s 不存在字段 %s，跳过清理", rule.TableName, rule.TimestampColumn)
			continue
		}

		// 清理该表的旧记录
		query := fmt.Sprintf("DELETE FROM %s WHERE %s < ?", rule.TableName, rule.TimestampColumn)
		result, err := al.DbHelper.db.Exec(query, cutoff)
		if err != nil {
			log.Printf("清理表 %s 失败: %v", rule.TableName, err)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("从表 %s 清理了 %d 条超过%v的记录", rule.TableName, rowsAffected, rule.RetentionPeriod)
			totalDeleted += rowsAffected
		}
	}

	if totalDeleted > 0 {
		log.Printf("总共清理了 %d 条旧记录", totalDeleted)
	}

	return nil
}

// cleanupOldRecordsAuto 自动发现并清理所有表中超过1天的记录
func (al *AlgorithmLogger) cleanupOldRecordsAuto() error {
	cutoff := time.Now().Add(-12 * time.Hour)

	// 获取数据库中所有表名
	tables, err := al.getAllTables()
	if err != nil {
		return fmt.Errorf("获取表列表失败: %v", err)
	}

	totalDeleted := int64(0)

	// 遍历每个表进行清理
	for _, tableName := range tables {
		// 检查表是否有时间戳字段
		timestampColumn := al.getTimestampColumn(tableName)
		if timestampColumn == "" {
			log.Printf("表 %s 没有时间戳字段，跳过清理", tableName)
			continue
		}

		// 清理该表的旧记录
		query := fmt.Sprintf("DELETE FROM %s WHERE %s < ?", tableName, timestampColumn)
		result, err := al.DbHelper.db.Exec(query, cutoff)
		if err != nil {
			log.Printf("清理表 %s 失败: %v", tableName, err)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("从表 %s 清理了 %d 条超过1天的记录", tableName, rowsAffected)
			totalDeleted += rowsAffected
		}
	}

	if totalDeleted > 0 {
		log.Printf("总共清理了 %d 条超过1天的记录", totalDeleted)
	}

	return nil
}

// tableExists 检查表是否存在
func (al *AlgorithmLogger) tableExists(tableName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
	var name string
	err := al.DbHelper.db.QueryRow(query, tableName).Scan(&name)
	return err == nil
}

// columnExists 检查字段是否存在
func (al *AlgorithmLogger) columnExists(tableName, columnName string) bool {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := al.DbHelper.db.Query(query)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}

		if name == columnName {
			return true
		}
	}

	return false
}

// getAllTables 获取数据库中所有表名
func (al *AlgorithmLogger) getAllTables() ([]string, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`
	rows, err := al.DbHelper.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// getTimestampColumn 获取表中的时间戳字段名
func (al *AlgorithmLogger) getTimestampColumn(tableName string) string {
	// 常见的时间戳字段名
	commonTimestampFields := []string{"timestamp", "created_at", "updated_at", "time", "date_created"}

	// 获取表结构
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := al.DbHelper.db.Query(query)
	if err != nil {
		log.Printf("获取表 %s 结构失败: %v", tableName, err)
		return ""
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}

		// 检查是否为时间类型的字段
		if strings.ToUpper(dataType) == "DATETIME" || strings.ToUpper(dataType) == "TIMESTAMP" {
			columns = append(columns, name)
		}
	}

	// 优先选择常见的时间戳字段名
	for _, commonField := range commonTimestampFields {
		for _, column := range columns {
			if strings.ToLower(column) == commonField {
				return column
			}
		}
	}

	// 如果没有找到常见字段名，返回第一个时间类型字段
	if len(columns) > 0 {
		return columns[0]
	}

	return ""
}

// startCleanupTask 启动定时清理任务
func (al *AlgorithmLogger) startCleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := al.CleanupOldRecords(); err != nil {
				log.Printf("清理任务失败: %v", err)
			}
		}
	}
}

// Close 关闭数据库连接
func (al *AlgorithmLogger) Close() error {
	return al.DbHelper.Close()
}
