package screepssocket

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchPayload(t *testing.T) {
	testCases := []struct {
		Name string
		Data []byte
	}{
		{
			Name: "ProdMessage",
			Data: []byte(`["[\"user:66070d418fd0c2031b293da2/console\",{\"messages\":{\"log\":[\"saved\",\"\ud83c\udf4c Current tick CPU usage: 9.36470700000018\"],\"results\":[]}}]"]`),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := batchPayload(context.Background(), testCase.Data)
			require.NoError(t, err)
		})
	}

}
