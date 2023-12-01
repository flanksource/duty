package duty

import (
	"reflect"
	"testing"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		input  string
		output map[string]string
	}{
		{
			input:  "db=yourdb user=youruser cloudsql-instance-connection-name=yourconnectionname private_ip=true",
			output: map[string]string{"db": "yourdb", "user": "youruser", "cloudsql-instance-connection-name": "yourconnectionname", "private_ip": "true"},
		},
		{
			input:  "key1=value1 key2=value2 key3=value3",
			output: map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
		},
		// Add more test cases as needed
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := parseParams(test.input)

			if !reflect.DeepEqual(result, test.output) {
				t.Errorf("Expected %v, but got %v", test.output, result)
			}
		})
	}
}
