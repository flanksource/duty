package drivers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseParams", func() {
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
		test := test // capture range variable
		It("should parse "+test.input, func() {
			result := parseParams(test.input)
			Expect(result).To(Equal(test.output))
		})
	}
})
