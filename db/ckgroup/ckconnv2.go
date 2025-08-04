package ckgroup

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"reflect"
	"strings"
	"time"
)

var CKCONN ClickHouseClient

type Column struct {
	Name string
	Type string
}

// ClickHouseClient ClickHouse客户端
type ClickHouseClient struct {
	conn      driver.Conn
	db        *sql.DB
	batchSize int
}

// Config 配置结构
type Config struct {
	Hosts     string
	Database  string
	Username  string
	Password  string
	BatchSize int
	Debug     bool
}

// NewClickHouseClient 创建新的ClickHouse客户端
func NewClickHouseClient(config Config) (*ClickHouseClient, error) {
	// 使用原生连接
	addr := strings.Split(config.Hosts, ",")
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: addr,
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		Debug: config.Debug,
		Debugf: func(format string, v ...interface{}) {
			if config.Debug {
				fmt.Printf("[ClickHouse Debug] "+format+"\n", v...)
			}
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:      time.Second * 30,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// 测试连接
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	// 创建标准数据库连接用于查询
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: addr,
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
	})

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}
	ckconn := ClickHouseClient{
		conn:      conn,
		db:        db,
		batchSize: batchSize,
	}
	CKCONN = ckconn

	return &ckconn, nil
}

// Close 关闭连接
func (c *ClickHouseClient) Close() error {
	var err1, err2 error
	if c.conn != nil {
		err1 = c.conn.Close()
	}
	if c.db != nil {
		err2 = c.db.Close()
	}
	if err1 != nil {
		return err1
	}
	return err2
}

// BatchInsert 批量插入数据，支持nested结构
func (c *ClickHouseClient) BatchInsert(tableName string, data interface{}) error {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		return fmt.Errorf("data must be a slice")
	}

	dataLen := dataValue.Len()
	if dataLen == 0 {
		return nil
	}

	// 获取第一个元素来分析结构
	firstElem := dataValue.Index(0).Interface()
	columns, err := c.analyzeStructure(firstElem)
	if err != nil {
		return fmt.Errorf("failed to analyze data structure: %w", err)
	}

	// 分批处理数据
	for i := 0; i < dataLen; i += c.batchSize {
		end := i + c.batchSize
		if end > dataLen {
			end = dataLen
		}

		batch, err := c.prepareBatch(tableName, columns)
		if err != nil {
			return fmt.Errorf("failed to prepare batch: %w", err)
		}

		// 添加数据到批次
		for j := i; j < end; j++ {
			item := dataValue.Index(j).Interface()
			values, err := c.extractValues(item, columns)
			if err != nil {
				return fmt.Errorf("failed to extract values from item %d: %w", j, err)
			}

			if err := batch.Append(values...); err != nil {
				return fmt.Errorf("failed to append data to batch: %w", err)
			}
		}

		// 发送批次
		if err := batch.Send(); err != nil {
			return fmt.Errorf("failed to send batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// prepareBatch 准备批次
func (c *ClickHouseClient) prepareBatch(tableName string, columns []string) (driver.Batch, error) {
	sql := fmt.Sprintf("INSERT INTO %s (%s)", tableName, strings.Join(columns, ", "))
	return c.conn.PrepareBatch(context.Background(), sql)
}

// analyzeStructure 分析数据结构
func (c *ClickHouseClient) analyzeStructure(sample interface{}) ([]string, error) {
	v := reflect.ValueOf(sample)
	t := reflect.TypeOf(sample)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data must be struct or pointer to struct")
	}

	var columns []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 跳过私有字段
		if !field.IsExported() {
			continue
		}

		// 获取字段名
		columnName := c.getColumnName(field)
		if columnName == "-" {
			continue
		}

		columns = append(columns, columnName)
	}

	return columns, nil
}

// getColumnName 获取列名
func (c *ClickHouseClient) getColumnName(field reflect.StructField) string {
	if tag := field.Tag.Get("db"); tag != "" {
		return strings.Split(tag, ",")[0]
	}
	if tag := field.Tag.Get("json"); tag != "" {
		return strings.Split(tag, ",")[0]
	}
	return strings.ToLower(field.Name)
}

// extractValues 提取值
func (c *ClickHouseClient) extractValues(item interface{}, columns []string) ([]interface{}, error) {
	v := reflect.ValueOf(item)
	t := reflect.TypeOf(item)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	var values []interface{}
	columnIndex := 0

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !field.IsExported() {
			continue
		}

		columnName := c.getColumnName(field)
		if columnName == "-" {
			continue
		}

		if columnIndex >= len(columns) {
			break
		}

		value := c.convertValue(fieldValue)
		values = append(values, value)
		columnIndex++
	}

	return values, nil
}

// convertValue 转换值
func (c *ClickHouseClient) convertValue(fieldValue reflect.Value) interface{} {
	if !fieldValue.IsValid() {
		return nil
	}

	switch fieldValue.Kind() {
	case reflect.Slice, reflect.Array:
		// 处理数组/切片类型，包括nested结构
		if fieldValue.Len() == 0 {
			return []interface{}{}
		}

		elemType := fieldValue.Type().Elem()
		if elemType.Kind() == reflect.Struct {
			// 处理nested结构体数组
			var result []map[string]interface{}
			for i := 0; i < fieldValue.Len(); i++ {
				elem := fieldValue.Index(i)
				elemMap := make(map[string]interface{})
				elemType := elem.Type()

				for j := 0; j < elem.NumField(); j++ {
					if !elemType.Field(j).IsExported() {
						continue
					}
					fieldName := c.getColumnName(elemType.Field(j))
					if fieldName != "-" {
						elemMap[fieldName] = c.convertValue(elem.Field(j))
					}
				}
				result = append(result, elemMap)
			}
			return result
		} else {
			// 基础类型数组
			return fieldValue.Interface()
		}
	case reflect.Ptr:
		if fieldValue.IsNil() {
			return nil
		}
		return c.convertValue(fieldValue.Elem())
	default:
		return fieldValue.Interface()
	}
}

// Query 执行查询
func (c *ClickHouseClient) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(context.Background(), query, args...)
}

// QueryRow 执行单行查询
func (c *ClickHouseClient) QueryRow(query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(context.Background(), query, args...)
}

// QueryToStruct 查询并映射到结构体切片
func (c *ClickHouseClient) QueryToStruct(dest interface{}, query string, args ...interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	sliceValue := destValue.Elem()
	elemType := sliceValue.Type().Elem()

	// 如果元素类型是指针，获取实际类型
	actualElemType := elemType
	isPtr := false
	if elemType.Kind() == reflect.Ptr {
		actualElemType = elemType.Elem()
		isPtr = true
	}

	if actualElemType.Kind() != reflect.Struct {
		return fmt.Errorf("slice elements must be structs")
	}

	rows, err := c.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		// 创建新的结构体实例
		var newElem reflect.Value
		if isPtr {
			newElem = reflect.New(actualElemType)
		} else {
			newElem = reflect.New(actualElemType).Elem()
		}

		// 准备扫描目标
		scanDest := make([]interface{}, len(columns))
		structValue := newElem
		if isPtr {
			structValue = newElem.Elem()
		}

		for i, col := range columns {
			field := c.findStructField(structValue, col)
			if field.IsValid() && field.CanSet() {
				scanDest[i] = field.Addr().Interface()
			} else {
				var dummy interface{}
				scanDest[i] = &dummy
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return err
		}

		// 添加到切片
		if isPtr {
			sliceValue.Set(reflect.Append(sliceValue, newElem))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, newElem))
		}
	}

	return rows.Err()
}

// findStructField 查找结构体字段
func (c *ClickHouseClient) findStructField(structValue reflect.Value, columnName string) reflect.Value {
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldColumnName := c.getColumnName(field)
		if fieldColumnName == columnName {
			return structValue.Field(i)
		}
	}

	return reflect.Value{}
}

// Exec 执行SQL语句
func (c *ClickHouseClient) Exec(query string, args ...interface{}) error {
	return c.conn.Exec(context.Background(), query, args...)
}

// Count 获取表记录数
func (c *ClickHouseClient) Count(tableName string, where string, args ...interface{}) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if where != "" {
		query += " WHERE " + where
	}

	var count int64
	err := c.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (c *ClickHouseClient) CreateTable(database, table, order, desc string, cols []Column) error {
	hasCreatedAt := false
	for _, col := range cols {
		if col.Name == "created_at" {
			hasCreatedAt = true
			break
		}
	}
	if !hasCreatedAt {
		return fmt.Errorf("created_at column is required for table %s.%s", database, table)
	}

	if err := c.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s ON CLUSTER bms_cluster", database)); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s ON CLUSTER bms_cluster (\n", database, table))
	for i, col := range cols {
		sb.WriteString(fmt.Sprintf("  %s %s", col.Name, col.Type))
		if i < len(cols)-1 {
			sb.WriteString(",\n")
		}
	}
	sb.WriteString(fmt.Sprintf("\n)\nENGINE = ReplicatedMergeTree('/clickhouse/tables/%s/{shard}/%s', '{replica}')\n", database, table))
	sb.WriteString("PARTITION BY toYYYYMM(created_at)\n")
	sb.WriteString(fmt.Sprintf("ORDER BY (%s, intHash64(created_at))\n", order))
	sb.WriteString("SAMPLE BY intHash64(created_at)\n")
	sb.WriteString("SETTINGS index_granularity = 8192\n")
	sb.WriteString(fmt.Sprintf("COMMENT '%s';", desc))

	return c.Exec(sb.String())
}
func (c *ClickHouseClient) CreateDistributedTable(distDB, localTable, desc string, cols []Column) error {
	if len(cols) == 0 {
		return fmt.Errorf("columns must be provided")
	}

	if err := c.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s ON CLUSTER bms_cluster", distDB)); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s ON CLUSTER bms_cluster (\n", distDB, localTable+"_distributed"))

	for i, col := range cols {
		sb.WriteString(fmt.Sprintf("  %s %s", col.Name, col.Type))
		if i < len(cols)-1 {
			sb.WriteString(",\n")
		}
	}

	sb.WriteString(fmt.Sprintf("\n)\nENGINE = Distributed('bms_cluster', '%s', '%s')\n", distDB, localTable))
	sb.WriteString(fmt.Sprintf("COMMENT '%s';", desc))

	return c.Exec(sb.String())
}
