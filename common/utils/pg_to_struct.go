package utils

import (
	"bufio"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"strings"
)

var FindColumnsSql = `
SELECT
    a.attnum AS column_number,
    a.attname AS column_name,
    --format_type(a.atttypid, a.atttypmod) AS column_type,
    a.attnotnull AS not_null,
    COALESCE(pg_get_expr(ad.adbin, ad.adrelid), '') AS default_value,
    COALESCE(ct.contype = 'p', false) AS  is_primary_key,
    CASE
        WHEN a.atttypid = ANY ('{int,int8,int2}'::regtype[])
          AND EXISTS (
             SELECT 1 FROM pg_attrdef ad
             WHERE  ad.adrelid = a.attrelid
             AND    ad.adnum   = a.attnum
             AND    ad.adsrc = 'nextval('''
                || (pg_get_serial_sequence (a.attrelid::regclass::text
                                          , a.attname))::regclass
                || '''::regclass)'
             )
            THEN CASE a.atttypid
                    WHEN 'int'::regtype  THEN 'serial'
                    WHEN 'int8'::regtype THEN 'bigserial'
                    WHEN 'int2'::regtype THEN 'smallserial'
                 END
        WHEN a.atttypid = ANY ('{uuid}'::regtype[]) AND COALESCE(pg_get_expr(ad.adbin, ad.adrelid), '') != ''
            THEN 'autogenuuid'
        ELSE format_type(a.atttypid, a.atttypmod)
    END AS column_type
FROM pg_attribute a
JOIN ONLY pg_class c ON c.oid = a.attrelid
JOIN ONLY pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_constraint ct ON ct.conrelid = c.oid
AND a.attnum = ANY(ct.conkey) AND ct.contype = 'p'
LEFT JOIN pg_attrdef ad ON ad.adrelid = c.oid AND ad.adnum = a.attnum
WHERE a.attisdropped = false
AND n.nspname = 'public'
AND c.relname = ?
AND a.attnum > 0
ORDER BY a.attnum
`
var findTablesSql = `
SELECT
c.relkind AS type,
c.relname AS table_name
FROM pg_class c
JOIN ONLY pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public'
AND c.relkind = 'r'
ORDER BY c.relname
`

type Table struct {
	TableName string `gorm:"column:table_name"` //table name
}
type Column struct {
	ColumnNumber int    `gorm:"column_number"` // column index
	ColumnName   string `gorm:"column_name"`   // column_name
	ColumnType   string `gorm:"column_type"`   // column_type
}

// dataSource = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s password=%s","localhost", "5432", "postgres", "test", "disable", "123")
func FindTables(dataSource string) []Table {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println(fmt.Sprintf("recover from a fatal error : %v", e))
		}
	}()
	db, err := gorm.Open("postgres", dataSource)
	db.SingularTable(true)
	db.LogMode(true)
	if err != nil {
		panic(err)
	}
	var tables = make([]Table, 0, 10)
	db.Raw(findTablesSql).Find(&tables)
	return tables
}

// find columns' property by specific dataSource and table name
func FindColumns(dataSource string, tableName string) []Column {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println(fmt.Sprintf("recover from a fatal error : %v", e))
		}
	}()
	db, err := gorm.Open("postgres", dataSource)
	db.SingularTable(true)
	db.LogMode(true)
	if err != nil {
		panic(err)
	}
	var columns = make([]Column, 0, 10)
	db.Raw(FindColumnsSql, tableName).Find(&columns)
	return columns
}

// generate go model y specific the dataSource and table name
func TableToStruct(dataSource string, tableName string) string {

	columnString := ""
	tmp := ""
	columns := FindColumns(dataSource, tableName)
	for _, column := range columns {

		tmp = fmt.Sprintf("    %s  %s\n", column.ColumnName, typeConvert(column.ColumnType))
		columnString = columnString + tmp
	}

	rs := fmt.Sprintf("type %s struct{\n%s}", UnderLineToHump(HumpToUnderLine(tableName)), columnString)
	return rs
}

// AddJSONFormGormTag ??????json??????
func AddJSONFormGormTag(in string) string {
	var result string
	scanner := bufio.NewScanner(strings.NewReader(in))
	var oldLineTmp = ""
	var lineTmp = ""
	var propertyTmp = ""
	var seperateArr []string
	for scanner.Scan() {
		oldLineTmp = scanner.Text()
		lineTmp = strings.Trim(scanner.Text(), " ")
		if strings.Contains(lineTmp, "{") || strings.Contains(lineTmp, "}") {
			result = result + oldLineTmp + "\n"
			continue
		}
		seperateArr = Split(lineTmp, " ")
		// ?????????????????????????????????tag, ??????tag?????????tag
		if len(seperateArr) == 1 || len(seperateArr) == 3 {
			continue
		}
		propertyTmp = HumpToUnderLine(seperateArr[0])
		oldLineTmp = oldLineTmp + fmt.Sprintf("    `gorm:\"column:%s\" json:\"%s\" form:\"%s\"`", propertyTmp, propertyTmp, propertyTmp)
		result = result + oldLineTmp + "\n"
	}
	return result
}

// Split ?????????Split??????  a,,,,,,,b,,c     ???","???????????????[a,b,c]
func Split(s string, sub string) []string {
	var rs = make([]string, 0, 20)
	tmp := ""
	Split2(s, sub, &tmp, &rs)
	return rs
}

// Split2 ?????????Split??????????????????
func Split2(s string, sub string, tmp *string, rs *[]string) {
	s = strings.Trim(s, sub)
	if !strings.Contains(s, sub) {
		*tmp = s
		*rs = append(*rs, *tmp)
		return
	}
	for i := range s {
		if string(s[i]) == sub {
			*tmp = s[:i]
			*rs = append(*rs, *tmp)
			s = s[i+1:]
			Split2(s, sub, tmp, rs)
			return
		}
	}
}

// FindUpperElement ???????????????????????????????????????,?????????HumpToUnderLine
func FindUpperElement(s string) []string {
	var rs = make([]string, 0, 10)
	for i := range s {
		if s[i] >= 65 && s[i] <= 90 {
			rs = append(rs, string(s[i]))
		}
	}
	return rs
}

// HumpToUnderLine ??????????????????
func HumpToUnderLine(s string) string {
	if s == "ID" {
		return "id"
	}
	var rs string
	elements := FindUpperElement(s)
	for _, e := range elements {
		s = strings.Replace(s, e, "_"+strings.ToLower(e), -1)
	}
	rs = strings.Trim(s, " ")
	rs = strings.Trim(rs, "\t")
	return strings.Trim(rs, "_")
}

// UnderLineToHump ??????????????????
func UnderLineToHump(s string) string {
	arr := strings.Split(s, "_")
	for i, v := range arr {
		arr[i] = strings.ToUpper(string(v[0])) + string(v[1:])
	}
	return strings.Join(arr, "")
}

// ????????????pg->go
func typeConvert(s string) string {
	if strings.Contains(s, "char") || in(s, []string{
		"text",
	}) {
		return "string"
	}
	if in(s, []string{"bigint", "bigserial", "integer", "smallint", "serial", "big serial"}) {
		return "int"
	}
	if in(s, []string{"numeric", "decimal", "real"}) {
		return "decimal.Decimal"
	}
	if in(s, []string{"bytea"}) {
		return "[]byte"
	}
	if strings.Contains(s, "time") || in(s, []string{"date"}) {
		return "time.Time"
	}
	if in(s, []string{"bigint", "bigserial", ""}) {
		return "json.RawMessage"
	}
	return "interface{}"
}

// ??????
func in(s string, arr []string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

func main() {
	dataSouce := fmt.Sprintf("host=112.16.6.211 user=soeuser port=9999 dbname=soedaturadb sslmode=disable password=soesoft application_name=syspara-rest")
	// ???????????????????????????????????????
	tables := FindTables(dataSouce)
	fmt.Println(tables)

	// ???????????????????????????????????????????????????
	tableName := "member_info"
	columns := FindColumns(dataSouce, tableName)
	fmt.Println(columns)

	// ??????????????????????????????go?????????
	goModel := TableToStruct(dataSouce, tableName)
	fmt.Println(goModel)
	// ?????????tag????????????
	goModelWithTag := AddJSONFormGormTag(goModel)
	fmt.Println(goModelWithTag)
}
