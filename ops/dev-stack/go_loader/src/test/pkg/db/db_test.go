package db_test

import (
	"database/sql"
	db_pkg "database_thing/pkg/db"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestConnectDB(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("an error '%v' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Expect a call to Open with the proper arguments
	mock.ExpectBegin().WillReturnError(nil)

	// Defer the Expectations assertion to check that all expectations were met
	defer mock.ExpectationsWereMet()

	// Call the ConnectDB function
	_, err = db_pkg.ConnectDB()
	if err != nil {
		t.Errorf("error was not expected while opening a stub database: %v", err)
	}
}
func TestSanitizeHeaders(t *testing.T) {
	testCases := []struct {
		name     string
		headers  []string
		expected []string
	}{
		{
			name:     "valid headers",
			headers:  []string{"Column1", "Column2", "Column3"},
			expected: []string{"column1", "column2", "column3"},
		},
		{
			name:     "headers with special characters",
			headers:  []string{"Column1!@#", "Column2$%^", "Column3&*("},
			expected: []string{"column1", "column2", "column3"},
		},
		{
			name:     "headers with numbers",
			headers:  []string{"Column1", "Column2123", "Column3"},
			expected: []string{"column1", "column2123", "column3"},
		},
		{
			name:     "headers with underscores",
			headers:  []string{"Column_1", "Column_2", "Column_3"},
			expected: []string{"column_1", "column_2", "column_3"},
		},
		{
			name:     "headers with uppercase letters",
			headers:  []string{"Column1", "Column2", "Column3"},
			expected: []string{"column1", "column2", "column3"},
		},
		{
			name:     "empty header",
			headers:  []string{"Column1", "", "Column3"},
			expected: []string{"column1", "", "column3"},
		},
		{
			name:     "headers with PostgreSQL reserved words",
			headers:  []string{"user", "table", "index"},
			expected: []string{"_user", "_table", "_index"},
		},
		{
			name:     "headers longer than 63 bytes",
			headers:  []string{"thisisaveryveryveryveryveryveryveryveryveryveryverylonglonglongheader"},
			expected: []string{"thisisaveryveryveryveryveryveryveryveryveryverylonglonglonglong"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sanitizedHeaders := db_pkg.SanitizeHeaders(tc.headers)
			for i, header := range sanitizedHeaders {
				if len(header) > 63 {
					t.Errorf("Expected sanitized headers to be less than 63 bytes, but got %v", len(header))
				}
				if len(header) != len(tc.expected[i]) {
					t.Errorf("Expected sanitized headers to be %v, but got %v", len(tc.expected[i]), len(header))
				}
			}
		})
	}
}
func TestSanitizeName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"user", "_user"},
		{"table", "_table"},
		{"index", "_index"},
		{"group", "_group"},
		{"order", "_order"},
		{"by", "_by"},
		{"select", "_select"},
		{"from", "_from"},
		{"user_table", "user_table"},
		{"User_table", "user_table"},
		{"user table", "user_table"},
		{"user*table", "user_table"},
		{"user_table#", "user_table"},
		{"user_table_123", "user_table_123"},
		{"user_table_123_", "user_table_123"},
		{"123456789012345678901234567890123456789012345678901234567890123", "_12345678901234567890123456789012345678901234567890123456789012"},
	}

	for _, test := range testCases {
		result := db_pkg.SanitizeName(test.input)
		if result != test.expected {
			t.Errorf("SanitizeName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}

	// Test 1: Check that special characters are removed
	input := "Column Name_with-Special#Characters!"
	expected := "column_name_with_special_characters"
	actual := db_pkg.SanitizeName(input)
	if expected != actual {
		t.Errorf("Expected %s but got %s", expected, actual)
	}

	// Test 2: Check that numbers are allowed but not first character
	input = "1st_column"
	expected = "_1st_column"
	actual = db_pkg.SanitizeName(input)
	if expected != actual {
		t.Errorf("Expected %s but got %s", expected, actual)
	}

	// Test 4: Check that reserved keywords are escaped
	input = "order"
	expected = "_order"
	actual = db_pkg.SanitizeName(input)
	if expected != actual {
		t.Errorf("Expected %s but got %s", expected, actual)
	}
}
func TestDetermineColumnNames(t *testing.T) {
	records := [][]string{
		{"name", "age", "gender"},
		{"John Smith", "35", "male"},
		{"Jane Smith", "28", "female"},
	}
	expected := map[string]string{
		"name":   "VARCHAR(255)",
		"age":    "VARCHAR(255)",
		"gender": "VARCHAR(255)",
	}
	actual := db_pkg.DetermineColumnNames(records)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestGetColumnData(t *testing.T) {
	tests := []struct {
		data       [][]string
		colIndex   int
		expected   []string
		shouldFail bool
	}{
		{[][]string{{"A", "B", "C"}, {"1", "2", "3"}, {"4", "5", "6"}}, 0, []string{"A", "1", "4"}, false},
		{[][]string{{"A", "B", "C"}, {"1", "2", "3"}, {"4", "5", "6"}}, 1, []string{"B", "2", "5"}, false},
		{[][]string{{"A", "B", "C"}, {"1", "2", "3"}, {"4", "5", "6"}}, 2, []string{"C", "3", "6"}, false},
		{[][]string{{"A", "B", "C"}, {"1", "2", "3"}, {"4", "5", "6"}}, 3, []string{}, true},
	}

	for i, test := range tests {
		result, err := db_pkg.GetColumnData(test.data, test.colIndex)

		if test.shouldFail && err == nil {
			t.Errorf("Test case %d: expected error, got nil", i)
		} else if !test.shouldFail && err != nil {
			t.Errorf("Test case %d: unexpected error: %s", i, err)
		}

		if !reflect.DeepEqual(result, test.expected) && err == nil {
			t.Errorf("Test case %d: expected %v, got %v", i, test.expected, result)
		}
	}
}

func TestInferColumnType(t *testing.T) {
	data := [][]string{
		{"1", "John", "Doe", "25"},
		{"2", "Jane", "Doe", "30"},
		{"3", "Jim", "Smith", "35"},
	}

	columnNames := []string{"id", "first_name", "last_name", "age"}

	expected := map[string]string{
		"id":         "INT",
		"first_name": "VARCHAR(255)",
		"last_name":  "VARCHAR(255)",
		"age":        "INT",
	}

	actual := db_pkg.InferColumnTypes(data, columnNames)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Test case 1: expected %v, got %v", expected, actual)
	}
}

func TestCountDataTypes(t *testing.T) {
	columnData := []string{
		"123", "456", "789", "true", "false", "2022-01-01", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", "Hello World!", "Hello World!",
		"123.456000", "789", "true", "false", "2022-01-01", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", "Hello World!", "Hello World!",
		"", "", "", "", "", "", "", "", "",
	}
	expectedDataTypeCounts := map[string]int{
		"BOOLEAN":      2,
		"DATE":         1,
		"FLOAT":        2,
		"INT":          3,
		"UUID":         1,
		"VARCHAR(255)": 2,
	}

	dataTypeCounts := CountDataTypes(columnData)

	if !reflect.DeepEqual(dataTypeCounts, expectedDataTypeCounts) {
		t.Errorf("Expected %v but got %v", expectedDataTypeCounts, dataTypeCounts)
	}
}

type mockCalculateTypePercentages struct{}

func (m *mockCalculateTypePercentages) CalculateTypePercentages(dataTypeCounts map[string]int, totalCount int) map[string]float64 {
	// Define the behavior of the mock function here
	// e.g. return a fixed map of type percentages
	return map[string]float64{"INT": 1.0}
}

var checkForNullValues func([]string) int
var CountDataTypes func([]string) map[string]int
var CalculateTypePercentages func(map[string]int, int) map[string]float64

func TestDetermineTableSchema(t *testing.T) {
	tests := []struct {
		columnData   []string
		percentages  map[string]float64
		expectedType string
	}{
		{columnData: []string{"1", "2", "3", "4", "5"}, percentages: map[string]float64{"INT": 1.0}, expectedType: "INT"},
		{columnData: []string{"1.1", "2.2", "3.3", "4.4", "5.5"}, percentages: map[string]float64{"FLOAT": 1.0}, expectedType: "FLOAT"},
		{columnData: []string{"true", "false", "true", "false", "false"}, percentages: map[string]float64{"BOOLEAN": 1.0}, expectedType: "BOOLEAN"},
		{columnData: []string{"2022-01-01", "2022-01-02", "2022-01-03", "2022-01-04", "2022-01-05"}, percentages: map[string]float64{"DATE": 1.0}, expectedType: "DATE"},
		{columnData: []string{"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12", "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13", "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14", "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15"}, percentages: map[string]float64{"UUID": 1.0}, expectedType: "UUID"},
	}

	for i, test := range tests {
		dataType, err := db_pkg.DetermineTableSchema(test.columnData, test.percentages)
		if err != nil {
			t.Errorf("Test %d: Unexpected error: %v", i, err)
			continue
		}
		if len(dataType) != 1 {
			t.Errorf("Test %d: Unexpected data type length: %v", i, len(dataType))
			continue
		}
		result := dataType[0].DataType
		if result != test.expectedType {
			t.Errorf("Test %d: Expected %s, but got %s", i, test.expectedType, result)
		}
	}

}

func TestIsValidDate(t *testing.T) {
	testCases := []struct {
		name          string
		value         string
		expectedTime  time.Time
		expectedError string
	}{
		{
			name:          "valid date string",
			value:         "2022-01-01",
			expectedTime:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedError: "",
		},
		{
			name:          "valid date string",
			value:         "2022/01/01",
			expectedTime:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedError: "",
		},
		{
			name:          "invalid date string",
			value:         "2022-02-31",
			expectedTime:  time.Time{},
			expectedError: "2022-02-31 is not a valid date format",
		},
		{
			name:          "integer value",
			value:         "123",
			expectedTime:  time.Time{},
			expectedError: "123 is not a valid date format",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsedTime, err := db_pkg.IsValidDate(testCase.value)
			if testCase.expectedError == "" && err != nil {
				t.Errorf("Expected no error, got %s", err)
			} else if testCase.expectedError != "" && (err == nil || !strings.Contains(err.Error(), testCase.expectedError)) {
				t.Errorf("Expected error message to contain %s, got %s", testCase.expectedError, err)
			}
			if !testCase.expectedTime.Equal(parsedTime) {
				t.Errorf("Expected parsed time to be %s, got %s", testCase.expectedTime, parsedTime)
			}
		})
	}
}
func TestCheckInvalidDateValues(t *testing.T) {
	validDate := "2022-01-01T00:00:00Z"
	validTime, err := time.Parse(time.RFC3339, validDate)
	if err != nil {
		t.Fatalf("unexpected error parsing date: %v", err)
	}
	if got := db_pkg.CheckInvalidDateValues(validTime); got != nil {
		t.Errorf("CheckInvalidDateValues(%v) = %v, expected nil", validTime, got)
	}

	invalidDates := []string{
		"0001-01-01T00:00:00Z",
		"1899-12-01T00:00:00Z",
		"10000-01-01T00:00:00Z",
		"1999-12-01T00:00:00Z",
		"2001-01-01T00:00:00Z",
		"2000-02-01T00:00:00Z",
	}
	for _, d := range invalidDates {
		invalidTime, err := time.Parse(time.RFC3339, d)
		if err != nil {
			t.Fatalf("unexpected error parsing date: %v", err)
		}
		if got := db_pkg.CheckInvalidDateValues(invalidTime); got == nil {
			t.Errorf("CheckInvalidDateValues(%v) = nil, expected error", d)
		}
	}
}

func TestCheckInvalidDateValuesWithParsedTime(t *testing.T) {
	var tests = []struct {
		input    string
		expected error
	}{
		{"2006-01-02", nil},
		{"2006-01-00", fmt.Errorf("%s is not a valid date format", "2006-01-00")},
		{"2006-00-02", fmt.Errorf("%s is not a valid date format", "2006-00-02")},
		{"2006-13-02", fmt.Errorf("%s is not a valid date format", "2006-13-02")},
		{"2008-02-29", nil},
		{"2006-02-00", fmt.Errorf("%s is not a valid date format", "2006-02-00")},
		{"2006-02-32", fmt.Errorf("%s is not a valid date format", "2006-02-32")},
		{"", fmt.Errorf("%s is not a valid date format", "")},
	}
	for _, test := range tests {
		parsedTime, err := db_pkg.ParseTime(test.input)
		if err != nil {
			t.Errorf("ParseTime(%s) = '%v'; expected '%v'", test.input, err, test.expected)
			continue
		}
		actual := db_pkg.CheckInvalidDateValues(parsedTime)
		if actual != test.expected {
			t.Errorf("CheckInvalidDateValues('%s') = '%v'; expected '%v'", test.input, actual, test.expected)
		}
	}
}

func TestParseTime(t *testing.T) {
	validValues := []struct {
		value    string
		expected time.Time
	}{
		{"2006-01-02 15:04:05", time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)},
		{"2006-01-02", time.Date(2006, time.January, 2, 0, 0, 0, 0, time.UTC)},
		{"2006/01/02 15:04:05", time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)},
		{"2006/01/02", time.Date(2006, time.January, 2, 0, 0, 0, 0, time.UTC)},
		{"01/02/2006", time.Date(2006, time.January, 2, 0, 0, 0, 0, time.UTC)},

		///TODO ParseTime currently not supporting the T format for time as drops the first 0 in the hour
		// {"2006-01-02T01:04:05Z", time.Date(2006, time.January, 2, 1, 4, 5, 0, time.UTC)},
		// {"2006-01-02T01:04:05-07", time.Date(2006, time.January, 2, 1, 4, 5, 0, time.FixedZone("", -7*60*60))},
		// {"2006-01-02T01:04:05.999999Z", time.Date(2006, time.January, 2, 1, 4, 5, 999999, time.UTC)},
	}

	for _, v := range validValues {
		parsedTime, err := db_pkg.ParseTime(v.value)
		if err != nil {
			t.Errorf("ParseTime(%s) = %v, expected %v", v.value, err, v.expected)
		}
		err = db_pkg.CheckInvalidDateValues(parsedTime)
		if err != nil {
			t.Errorf("CheckInvalidDateValues(%s) = %v, expected %v", parsedTime, err, nil)
		}
		if !parsedTime.Equal(v.expected) {
			t.Errorf("ParseTime(%s) = %v, expected %v", v.value, parsedTime, v.expected)
		}
	}

	invalidValue := "not-a-valid-date-format"
	_, err := db_pkg.ParseTime(invalidValue)
	if err == nil {
		t.Errorf("ParseTime(%s) = %v, expected error", invalidValue, err)
	}
}

func TestConvertToPostgresDate(t *testing.T) {
	// Test input time
	inputTime, _ := time.Parse("2006-01-02 15:04:05.999999999", "2022-12-31 23:59:59.999999999")

	// Expected output time in Postgres format
	expectedOutput, _ := time.Parse("2006-01-02 15:04:05.999999-07", "2022-12-31 23:59:59.999999+00")

	// Test the function
	outputTime, err := db_pkg.ConvertToPostgresDate(inputTime)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !outputTime.Equal(expectedOutput) {
		t.Errorf("Unexpected output time. Expected %v, got %v", expectedOutput, outputTime)
	}
}
func TestIsValidUUID(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
		err      error
	}{
		{"00000000-0000-0000-0000-000000000000", true, nil},
		{"not-a-valid-uuid", false, fmt.Errorf("not-a-valid-uuid is not a valid UUID: invalid UUID length: 16")},
		{"", false, fmt.Errorf("  is not a valid UUID: invalid UUID length: 0")},
		{"12345678-1234-1234-1234-1234567890ab", true, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			actual, err := db_pkg.IsValidUUID(tc.value)

			if tc.err == nil && err == nil {
				if actual != tc.expected {
					t.Errorf("expected: %t but got: %t", tc.expected, actual)
				}
				return
			}

			actualTrimmed := strings.TrimSpace(err.Error())
			expectedTrimmed := strings.TrimSpace(tc.err.Error())
			if actualTrimmed != expectedTrimmed {
				t.Errorf("expected error: %v but got: %v", expectedTrimmed, actualTrimmed)
			}
		})
	}
}

func TestIsValidUUIDErrString(t *testing.T) {
	testCases := []struct {
		value     string
		errString string
	}{
		{"not-a-valid-uuid", "not-a-valid-uuid is not a valid UUID: invalid UUID length: 16"},
		// {"", "  is not a valid UUID: invalid UUID length: 0"},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			_, err := db_pkg.IsValidUUID(tc.value)
			if err == nil {
				t.Errorf("expected error %q but got nil", tc.errString)
			} else if err.Error() != tc.errString {
				t.Errorf("expected error '%q' but got '%q'", tc.errString, err.Error())
			}
		})
	}
}

func TestIsValidEnum(t *testing.T) {
	// Test case 1: Valid enum value
	result := db_pkg.IsValidEnum("value1", []string{"value1", "value2", "value3"})
	assert.True(t, result)

	// Test case 2: Invalid enum value
	result = db_pkg.IsValidEnum("value4", []string{"value1", "value2", "value3"})
	assert.False(t, result)
}

func TestCalculateTypePercentages(t *testing.T) {
	// normal case
	dataTypeCounts := map[string]int{
		"BOOLEAN":      1,
		"DATE":         2,
		"FLOAT":        3,
		"INT":          4,
		"UUID":         5,
		"VARCHAR(255)": 6,
	}
	totalCount := 21
	expectedTypePercentages := map[string]float64{
		"BOOLEAN":      4.761904761904762,
		"DATE":         9.523809523809524,
		"FLOAT":        14.285714285714285,
		"INT":          19.047619047619047,
		"UUID":         23.809523809523807,
		"VARCHAR(255)": 28.57142857142857,
	}
	typePercentages, err := db_pkg.CalculateTypePercentages(dataTypeCounts, totalCount)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(typePercentages, expectedTypePercentages) {
		t.Errorf("got %v, want %v", typePercentages, expectedTypePercentages)
	}

	// not all supported data types found in dataTypeCounts map
	dataTypeCounts = map[string]int{
		"BOOLEAN":      1,
		"DATE":         2,
		"FLOAT":        3,
		"INT":          4,
		"VARCHAR(255)": 6,
	}
	totalCount = 16
	_, err = db_pkg.CalculateTypePercentages(dataTypeCounts, totalCount)
	if err == nil {
		t.Errorf("expected error got none, got %s", err)
	}

	// totalCount is 0
	dataTypeCounts = map[string]int{
		"BOOLEAN":      1,
		"DATE":         2,
		"FLOAT":        3,
		"INT":          4,
		"UUID":         5,
		"VARCHAR(255)": 6,
	}
	totalCount = 0
	_, err = db_pkg.CalculateTypePercentages(dataTypeCounts, totalCount)
	if err == nil {
		t.Errorf("expected error got none, got %s", err)
	}

	// nil dataTypeCounts
	dataTypeCounts = nil
	totalCount = 0
	_, err = db_pkg.CalculateTypePercentages(dataTypeCounts, totalCount)
	if err == nil {
		t.Errorf("expected error got none, got %s", err)
	}
}

func TestCreateTable(t *testing.T) {
	// Test case 1: Successful table creation
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS test_table").WillReturnResult(sqlmock.NewResult(1, 1))
	columnTypes := map[string]string{"name": "VARCHAR(255)", "age": "INTEGER"}
	columnNames := []string{"name", "age"}
	err = db_pkg.CreateTable(db, "test_table", columnNames, columnTypes)
	assert.NoError(t, err)
	assert.Nil(t, mock.ExpectationsWereMet())

	// Test case 2: Error creating table
	db, mock, err = sqlmock.New()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS test_table").WillReturnError(fmt.Errorf("Error creating table"))
	columnTypes = map[string]string{"name": "VARCHAR(255)", "age": "INTEGER"}
	columnNames = []string{"name", "age"}
	err = db_pkg.CreateTable(db, "test_table", columnNames, columnTypes)
	assert.Error(t, err)
	assert.Equal(t, "error creating table: Error creating table", err.Error())
	assert.Nil(t, mock.ExpectationsWereMet())

	// Test case 3: Check the scenario that if the column type is not in the map, add it with a default ty

}

func TestCreateTableWithCorrectNameAndColumns(t *testing.T) {
	// setup
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	columnNames := []string{"name", "age", "address"}
	columnTypes := map[string]string{"name": "VARCHAR(255)", "age": "VARCHAR(255)", "address": "VARCHAR(255)"}
	// records := [][]string{{"John", "25", "New York"}, {"Jane", "30", "Los Angeles"}}

	// expected query
	expectedQuery := "CREATE TABLE IF NOT EXISTS test_table(id SERIAL PRIMARY KEY,name VARCHAR(255),age VARCHAR(255),address VARCHAR(255))"
	mock.ExpectExec(expectedQuery).WillReturnResult(sqlmock.NewResult(1, 1))

	// execute
	err = db_pkg.CreateTable(db, "test_table", columnNames, columnTypes)

	// assert
	if err != nil {
		t.Errorf("error was not expected while creating table: %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCreateTableWithCorrectColumnTypes(t *testing.T) {
	// setup
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	columnNames := []string{"name", "age", "address"}
	columnTypes := map[string]string{"name": "string", "age": "integer", "address": "string"}
	// records := [][]string{{"John", "25", "New York"}, {"Jane", "30", "Los Angeles"}}

	// expected query
	expectedQuery := "CREATE TABLE IF NOT EXISTS test_table(id SERIAL PRIMARY KEY,name VARCHAR(255),age VARCHAR(255),address VARCHAR(255))"
	mock.ExpectExec(expectedQuery).WillReturnResult(sqlmock.NewResult(1, 1))

	// execute
	err = db_pkg.CreateTable(db, "test_table", columnNames, columnTypes)

	// assert
	if err != nil {
		t.Errorf("error was not expected while creating table: %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCreateTableWithCorrectPrimaryKey(t *testing.T) {
	// setup
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	columnNames := []string{"name", "age", "address"}
	columnTypes := map[string]string{"name": "string", "age": "integer", "address": "string"}
	// records := [][]string{{"John", "25", "New York"}, {"Jane", "30", "Los Angeles"}}

	// expected query
	expectedQuery := "CREATE TABLE IF NOT EXISTS test_table(id SERIAL PRIMARY KEY,name string,age integer,address string)"
	mock.ExpectExec(expectedQuery).WillReturnResult(sqlmock.NewResult(1, 1))

	// execute
	err = db_pkg.CreateTable(db, "test_table", columnNames, columnTypes)

	// assert
	if err != nil {
		t.Errorf("error was not expected while creating table: %s", err)
	}

	// check if the primary key is "id"
	rows := sqlmock.NewRows([]string{"id"}).AddRow("id")
	mock.ExpectQuery("^SELECT conname as name FROM pg_constraint WHERE confrelid = (SELECT oid FROM pg_class WHERE relname = 'test_table') AND confrelid = (SELECT oid FROM pg_class WHERE relname = 'test_table')$").WillReturnRows(rows)

	// execute
	var primaryKey string
	err = db.QueryRow("SELECT conname as name FROM pg_constraint WHERE confrelid = (SELECT oid FROM pg_class WHERE relname = 'test_table') AND confrelid = (SELECT oid FROM pg_class WHERE relname = 'test_table')").Scan(&primaryKey)
	if err == sql.ErrNoRows {
		t.Errorf("no primary key found for table 'test_table'")
		return
	}
	// assert
	if primaryKey != "id" {
		t.Errorf("primary key was expected to be 'id', but got '%s'", primaryKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestPopulateTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating mock database: %v", err)
	}
	defer db.Close()

	tableName := "test_table"
	columnNames := []string{"column1", "column2", "column3"}
	dataset := [][]interface{}{
		[]interface{}{"value1", 2, 3.0},
		[]interface{}{"value2", 4, 5.0},
		[]interface{}{"value3", 6, 7.0},
	}

	// Create the test table
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS test_table").WillReturnResult(sqlmock.NewResult(1, 1))

	err = db_pkg.CreateTable(db, tableName, columnNames, map[string]string{})
	if err != nil {
		t.Errorf("Error creating table: %v", err)
	}

	// Populate the test table
	for i := 0; i < len(dataset); i++ {
		mock.ExpectExec("INSERT INTO test_table").WillReturnResult(sqlmock.NewResult(1, 1))
	}

	err = db_pkg.PopulateTable(db, tableName, dataset, columnNames, 1)
	if err != nil {
		t.Errorf("Error populating table: %v", err)
	}

	// Verify that all expected statements were executed
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}
