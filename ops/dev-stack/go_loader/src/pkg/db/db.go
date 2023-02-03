package db

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	_ "github.com/lib/pq"
	"go.uber.org/fx"
)

var Module = fx.Module("db", fx.Provide(ConnectDB))

// DataTypeInfo holds information about a data type.
type DataTypeInfo struct {
	DataType         string
	Supported        bool
	Size             int
	Precision        int
	Scale            int
	DataTypeCounters int
}

var dataTypeMapping = map[string]DataTypeInfo{
	"BIGINT":           {DataType: "BIGINT", Size: 64, Supported: true, DataTypeCounters: 0},
	"BOOLEAN":          {DataType: "BOOLEAN", Supported: true, DataTypeCounters: 0},
	"DATE":             {DataType: "DATE", Supported: true, DataTypeCounters: 0},
	"DOUBLE PRECISION": {DataType: "DOUBLE PRECISION", Size: 64, Precision: 15, Supported: true, DataTypeCounters: 0},
	"DECIMAL":          {DataType: "DECIMAL", Precision: 18, Supported: true, DataTypeCounters: 0},
	"FLOAT":            {DataType: "FLOAT", Size: 64, Precision: 6, Supported: true, DataTypeCounters: 0},
	"INT":              {DataType: "INT", Size: 32, Supported: true, DataTypeCounters: 0},
	"INTERVAL":         {DataType: "INTERVAL", Supported: true, DataTypeCounters: 0},
	"NUMERIC":          {DataType: "NUMERIC", Precision: 18, Supported: true, DataTypeCounters: 0},
	"SMALLINT":         {DataType: "SMALLINT", Size: 16, Supported: true, DataTypeCounters: 0},
	"TIME":             {DataType: "TIME", Supported: true, DataTypeCounters: 0},
	"TIMESTAMP":        {DataType: "TIMESTAMP", Supported: true, DataTypeCounters: 0},
	"TIMESTAMPZ":       {DataType: "TIMESTAMP", Supported: true, DataTypeCounters: 0},
	"VARCHAR(255)":     {DataType: "VARCHAR(255)", Size: 255, Supported: true, DataTypeCounters: 0},
	"UUID":             {DataType: "UUID", Supported: true, DataTypeCounters: 0},
}

const (
	host     = "postgres-db"
	port     = 5432
	user     = "go_loader"
	dbname   = "postgres"
	password = "go_loader"
)

// Connect to the PostgreSQL database
func ConnectDB() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
		os.Exit(1)
	}
	return db, nil
}

func SanitizeHeaders(headers []string) []string {
	sanitizedHeaders := make([]string, 0, len(headers))
	for _, header := range headers {
		sanitizedHeader := SanitizeName(header)
		sanitizedHeaders = append(sanitizedHeaders, sanitizedHeader)
	}
	return sanitizedHeaders
}

func SanitizeName(columnName string) string {
	// replace spaces with underscores
	columnName = strings.Replace(columnName, " ", "_", -1)
	// remove any special characters
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")
	columnName = reg.ReplaceAllString(columnName, "_")
	columnName = strings.TrimSuffix(columnName, "_")
	// check if column name is empty return empty string
	if columnName == "" {
		return ""
	}
	// check if first character is number
	if unicode.IsNumber(rune(columnName[0])) {
		columnName = "_" + columnName
	}
	// convert to lowercase
	columnName = strings.ToLower(columnName)
	// check if column name is longer than 63 characters
	if len(columnName) > 63 {
		columnName = columnName[:63]
	}
	// check if column name is a reserved word in PostgreSQL
	reservedWords := map[string]bool{"user": true, "table": true, "index": true, "group": true, "order": true, "by": true, "select": true, "from": true}
	if reservedWords[columnName] {
		columnName = `_` + columnName
	}
	return columnName
}

func DetermineColumnNames(rows [][]string) map[string]string {
	columnNames := make(map[string]string)
	usedNames := make(map[string]bool)
	for _, val := range rows[0] {
		val = SanitizeName(val)
		if _, ok := columnNames[val]; !ok {
			columnNames[val] = "VARCHAR(255)"
			usedNames[val] = true
		} else {
			newVal := val
			i := 1
			for usedNames[newVal] {
				newVal = val + strconv.Itoa(i)
				i++
			}
			columnNames[newVal] = "VARCHAR(255)"
			usedNames[newVal] = true
		}
	}
	return columnNames
}

func GetColumnData(data [][]string, colIndex int) ([]string, error) {
	var columnData []string
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	if colIndex >= len(data[0]) {
		return nil, fmt.Errorf("column index %d is out of range. there are only %d columns in the data", colIndex, len(data[0]))
	}
	for i := range data {
		if len(data[i]) <= colIndex {
			return nil, fmt.Errorf("row %d does not have a value for column index %d", i, colIndex)
		}
		columnData = append(columnData, data[i][colIndex])
	}
	return columnData, nil
}

func updateCounter(counters map[string]DataTypeInfo, dataType string) {
	dataTypeInfo := counters[dataType]
	dataTypeInfo.DataTypeCounters++
	counters[dataType] = dataTypeInfo
}

func DetermineDataType(columnData []string) (map[string]DataTypeInfo, int, error) {

	if len(columnData) == 0 {
		return nil, 0, fmt.Errorf("column data is empty or not found")
	}
	counters := dataTypeMapping
	totalCount := 0

	for i, value := range columnData {
		totalCount++
		if value == "" {
			updateCounter(counters, "NULL")
			continue
		}

		if determined := determineTimeTypes(value, counters); determined {
			continue
		}

		if determined := determineNumberType(value, counters); determined {
			continue
		}
		if determined := determineTextType(value, counters); determined {
			continue
		}

		fmt.Printf("Error: Could not determine data type of value %s at index %d\n", value, i)

		return nil, 0, fmt.Errorf("Could not determine data type of value %s", value)
	}

	return counters, len(columnData), nil
}

func determineTextType(value string, counters map[string]DataTypeInfo) bool {
	if match, _ := regexp.MatchString(`^.{0,255}$`, value); match {
		updateCounter(counters, "VARCHAR(255)")
		return true
	} else if match, _ := regexp.MatchString(`^.{0,65535}$`, value); match {
		updateCounter(counters, "TEXT")
		return true
	}
	return false
}

func determineTimeTypes(value string, counters map[string]DataTypeInfo) bool {
	var timeFormats = []string{
		"15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999",
		time.RFC3339,
		time.RFC3339Nano,
		"15:04",
		"15:04:05.999999",
	}
	for _, format := range timeFormats {
		if _, err := time.Parse(format, value); err == nil {
			switch {
			case format == "15:04:05":
				dataTypeInfo := counters["TIME"]
				dataTypeInfo.DataTypeCounters++
				counters["TIME"] = dataTypeInfo
				return true
			case strings.HasSuffix(format, ":04:05"):
				dataTypeInfo := counters["TIME"]
				dataTypeInfo.DataTypeCounters++
				counters["TIME"] = dataTypeInfo
				return true
			case format == "2006-01-02 15:04:05.999999-07":
				dataTypeInfo := counters["TIMESTAMPZ"]
				dataTypeInfo.DataTypeCounters++
				counters["TIMESTAMPZ"] = dataTypeInfo
				return true
			default:
				dataTypeInfo := counters["TIME"]
				dataTypeInfo.DataTypeCounters++
				counters["TIME"] = dataTypeInfo
				return true
			}
		}
	}
	return false
}
func determineDecimalType(value string, counters map[string]DataTypeInfo) {

}

func determineCharType(value string, counters map[string]DataTypeInfo) {
	if match, _ := regexp.MatchString(`^.{1,1}$`, value); match {
		updateCounter(counters, "CHAR")
		return
	}
}

func determineByteaType(value string, counters map[string]DataTypeInfo) {
	if match, _ := regexp.MatchString(`^[\x00-\xff]*$`, value); match {
		updateCounter(counters, "BYTEA")
		return
	}
}

func determineNumberType(value string, counters map[string]DataTypeInfo) bool {
	if parsedInt, intErr := strconv.ParseInt(value, 10, 64); intErr == nil {
		if parsedInt >= math.MinInt32 && parsedInt <= math.MaxInt32 {
			updateCounter(counters, "INT")
			return true
		} else if parsedInt >= -32768 && parsedInt <= 32767 {
			updateCounter(counters, "SMALLINT")
			return true
		} else {
			updateCounter(counters, "BIGINT")
			return true
		}
	}

	var parsedFloat float64
	var floatErr error

	if parsedFloat, floatErr = strconv.ParseFloat(value, 32); floatErr == nil {
		updateCounter(counters, "DECIMAL")
		return true
	} else if parsedFloat, floatErr = strconv.ParseFloat(value, 64); floatErr == nil {
		if math.MaxInt64-parsedFloat < 1e-10 {
			updateCounter(counters, "NUMERIC")
			return true
		} else {
			updateCounter(counters, "FLOAT")
			return true
		}
	}

	if value == "true" || value == "false" {
		updateCounter(counters, "BOOL")
		return true
	}

	return false
}

func CalculateTypePercentages(dataTypeCounts map[string]DataTypeInfo, totalCount int) (map[string]float64, error) {

	if dataTypeCounts == nil {
		return nil, fmt.Errorf("dataTypeCounts map is nil")
	}
	typePercentages := make(map[string]float64)

	allDataTypesFound := true

	for _, dataTypeInfo := range dataTypeMapping {
		if dataTypeInfo.Supported {
			if counts, ok := dataTypeCounts[dataTypeInfo.DataType]; ok {
				typePercentages[dataTypeInfo.DataType] = float64(counts.DataTypeCounters) / float64(totalCount) * 100
			} else {
				allDataTypesFound = false
			}
		}
	}

	if !allDataTypesFound {
		return nil, fmt.Errorf("not all supported data types found in dataTypeCounts map")
	}
	if totalCount == 0 {
		return nil, fmt.Errorf("total count is 0")
	}
	return typePercentages, nil
}

// TODO: add logic to determine if column is a primary, foreign,unique, or composite key
// TODO: add minimum thresholds to determine
// DetermineTableSchema takes in a slice of column data and a map of type percentages,
// and returns a slice of data type information and an error if any.
func DetermineTableSchema(columnData []string, typePercentages map[string]float64, counters map[string]DataTypeInfo) ([]DataTypeInfo, error) {
	if len(columnData) == 0 {
		return nil, fmt.Errorf("column data is empty")
	}

	if len(counters) > 0 {
		for key := range counters {
			counters[key] = DataTypeInfo{DataType: key, Size: 0}
		}
	}

	counters, totalCount, err := DetermineDataType(columnData)
	if err != nil {
		return nil, fmt.Errorf("error in DetermineDataType: %v", err)
	}

	calculatedPercentages, err := CalculateTypePercentages(counters, totalCount)
	if err != nil {
		return nil, fmt.Errorf("error in calculatetypepercentages: %v", err)
	}
	fmt.Println("Calculated percentages:", calculatedPercentages)

	var matchingDataTypes []DataTypeInfo
	for dataType, _ := range typePercentages {
		percentage, ok := typePercentages[dataType]
		if !ok {
			return nil, fmt.Errorf("missing typePercentage for %s", dataType)
		}
		if calculatedPercentages[dataType] >= percentage {
			fmt.Println("Matching data type found:", dataType)
			matchingDataTypes = append(matchingDataTypes, DataTypeInfo{DataType: dataType, Size: 0})
		}
	}

	if len(matchingDataTypes) == 0 {
		fmt.Println("No matching data types found, returning default VARCHAR(255)")
		return []DataTypeInfo{{DataType: "VARCHAR(255)", Size: 255}}, nil
	}

	return matchingDataTypes, nil
}

func CheckForNullValues(columnData []string) (int, error) {
	nullCount := 0
	for i, value := range columnData {
		if len(value) == 0 {
			nullCount++
		} else if strings.ToLower(value) == "null" {
			return 0, fmt.Errorf("error in row %d: string value 'null' cannot be used in place of a null value", i+1)
		}
	}
	return nullCount, nil
}

func ParseTime(value string) (time.Time, error) {
	var parsedTime time.Time
	var err error
	formatStrings := []string{"2006-01-02 15:04:05", "2006-01-02", "2006/01/02 15:04:05", "2006/01/02", "01/02/2006", "2006-01-02T01:04:05Z", "2006-01-02T01:04:05-07", "2006-01-02T01:04:05.999999Z"}
	for _, format := range formatStrings {
		parsedTime, err = time.Parse(format, value)
		if err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("%s is not a valid date format", value)
	}
	return parsedTime, nil
}

func CheckInvalidDateValues(parsedTime time.Time) error {
	if parsedTime.IsZero() {
		return fmt.Errorf("%s is not a valid date format", parsedTime.String())
	}

	if parsedTime.Month() == 0 || parsedTime.Day() == 0 {
		return fmt.Errorf("%s is not a valid date format", parsedTime.String())
	}

	if parsedTime.Month() < 1 || parsedTime.Month() > 12 {
		return fmt.Errorf("%s is not a valid date format", parsedTime.String())
	}

	daysInMonth := 31
	if parsedTime.Month() == 2 {
		if parsedTime.Year()%4 == 0 && (parsedTime.Year()%100 != 0 || parsedTime.Year()%400 == 0) {
			daysInMonth = 29
		} else {
			daysInMonth = 28
		}
	} else if parsedTime.Month() == 4 || parsedTime.Month() == 6 || parsedTime.Month() == 9 || parsedTime.Month() == 11 {
		daysInMonth = 30
	}

	if parsedTime.Day() < 1 || parsedTime.Day() > daysInMonth {
		return fmt.Errorf("%s is not a valid date format", parsedTime.String())
	}
	return nil // no errors
}

func IsValidDate(value string) (time.Time, error) {
	// parse the time using helper function
	parsedTime, err := ParseTime(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s is not a valid date: %v", value, err)
	}

	// check invalid date values using helper function
	if err := CheckInvalidDateValues(parsedTime); err != nil {
		return parsedTime, fmt.Errorf("%s is not a valid date: %v", value, err)
	}

	// try to convert the value to the Postgres date format using the helper function
	parsedTime, err = ConvertToPostgresDate(parsedTime)
	if err != nil {
		return parsedTime, fmt.Errorf("%s unable to convert time format to: %v", value, err)
	}

	return parsedTime, nil
}

func ConvertToPostgresDate(parsedTime time.Time) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05.999999+07:00:00",
		"2006-01-02 15:04:05.999999+07:00",
		"2006-01-02 15:04:05.999999+07",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for i, format := range formats {
		formattedString := parsedTime.Format(format)
		parsedTime, err := time.Parse(format, formattedString)
		if err == nil {
			return parsedTime, nil
		} else {
			fmt.Printf("Error in format %d: %v\n", i, err)
		}
	}
	return time.Time{}, fmt.Errorf("could not convert %v to a Postgres compatible date format", parsedTime)
}

func IsValidBoolean(value string) bool {
	if value == "true" || value == "false" {
		return true
	}
	return false
}

func IsValidUUID(value string) (bool, error) {
	if _, err := uuid.Parse(value); err != nil {
		return false, fmt.Errorf("%s is not a valid UUID: %v", value, err)
	}
	return true, nil
}

func IsValidEnum(value string, possibleEnumValues []string) bool {
	// Check if the value is in the list of possible enum values
	for _, enumValue := range possibleEnumValues {
		if value == enumValue {
			return true
		}
	}
	return false
}

func InferColumnTypes(data [][]string, columnNames []string) map[string]string {
	if data == nil || len(data) < 2 || columnNames == nil || len(columnNames) == 0 {
		return nil
	}
	columnTypes := make(map[string]string)
	for colIndex, columnName := range columnNames {
		columnData, err := GetColumnData(data, colIndex)
		if err != nil {
			fmt.Println("Error: Unexpected error while getting column data:", err)
			columnTypes[columnName] = "VARCHAR(255)"
			continue
		}

		if len(columnData) == 0 {
			fmt.Println("Error: Column data is empty or not found")
			columnTypes[columnName] = "VARCHAR(255)"
			continue
		}

		dataTypeCounts, _, err := DetermineDataType(columnData)
		if err != nil {
			fmt.Println("Error:", err.Error())
			columnTypes[columnName] = "VARCHAR(255)"
			continue
		}
		if dataTypeCounts == nil || len(dataTypeCounts) == 0 {
			fmt.Println("Error: No data type counts found")
			columnTypes[columnName] = "VARCHAR(255)"
			continue
		}

		typePercentages, err := CalculateTypePercentages(dataTypeCounts, len(columnData))
		if err != nil {
			fmt.Println("Error:", err.Error())
			columnTypes[columnName] = "VARCHAR(255)"
			continue
		}

		tableSchema, err := DetermineTableSchema(columnData, typePercentages, dataTypeMapping)
		if err != nil {
			fmt.Println("Error:", err)
			columnTypes[columnName] = "VARCHAR(255)"
			continue
		}

		if len(tableSchema) == 0 {
			columnTypes[columnName] = "VARCHAR(255)"
			continue
		}

		dt := tableSchema[0].DataType
		columnTypes[columnName] = dt
		fmt.Println("Column Name:", columnName, "Data Type:", dt)
	}

	return columnTypes
}

func CreateTable(db *sql.DB, tableName string, columnNames []string, columnTypes map[string]string) error {
	if db == nil {
		return fmt.Errorf("error: Invalid database connection")
	}
	if len(columnNames) == 0 || len(columnTypes) == 0 {
		return fmt.Errorf("error: No columns provided")
	}
	// Prepare create table query
	query := "CREATE TABLE IF NOT EXISTS " + tableName + "("
	query += "id SERIAL PRIMARY KEY,"
	for i, col := range columnNames {
		// check if the column type is already in the map
		if _, ok := columnTypes[col]; !ok {
			// if it's not in the map, add it with a default type
			columnTypes[col] = "VARCHAR(255)"
		}
		query += col + " " + columnTypes[col]
		if i < len(columnNames)-1 {
			query += ","
		}
	}
	query += ")"
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating table: %v", err)
	}
	return nil
}

func PopulateTable(db *sql.DB, tableName string, dataset [][]interface{}, columnNames []string, batchSize int) error {
	if tableName == "" {
		return fmt.Errorf("tableName cannot be empty")
	}

	if len(dataset) == 0 {
		return fmt.Errorf("dataset cannot be empty")
	}

	if len(columnNames) == 0 {
		return fmt.Errorf("columnNames cannot be empty")
	}

	if batchSize <= 0 {
		return fmt.Errorf("batchSize must be greater than 0")
	}

	placeholders := make([]string, len(columnNames))
	for i := range placeholders {
		placeholders[i] = "$" + strconv.Itoa(i+1)
	}

	query := "INSERT INTO " + tableName + "("
	query += strings.Join(columnNames, ",")
	query += ") VALUES("
	query += strings.Join(placeholders, ",")
	query += ")"

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}

	defer stmt.Close()

	for i := 0; i < len(dataset); i += batchSize {
		end := i + batchSize
		if end > len(dataset) {
			end = len(dataset)
		}

		batch := dataset[i:end]
		for _, record := range batch {
			// Create a new slice for the arguments to Exec, using the order of the `columnNames` slice
			args := make([]interface{}, len(columnNames))
			for j, columnName := range columnNames {
				args[j] = record[j]
			}
			_, err = stmt.Exec(args...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// func GetAllData(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
// 	rows, err := db.Query("SELECT * FROM " + tableName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var data []map[string]interface{}
// 	columns, err := rows.Columns()
// 	if err != nil {
// 		return nil, err
// 	}

// 	values := make([]interface{}, len(columns))
// 	valuePtrs := make([]interface{}, len(columns))
// 	for i := range columns {
// 		valuePtrs[i] = &values[i]
// 	}

// 	for rows.Next() {
// 		err := rows.Scan(valuePtrs...)
// 		if err != nil {
// 			return nil, err
// 		}

// 		row := make(map[string]interface{})
// 		for i, col := range columns {
// 			val := values[i]
// 			row[col] = val
// 		}
// 		data = append(data, row)
// 	}
// 	return data, nil
// }
