package prometheusremotewritev2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestV2WriteRequestBuilder(t *testing.T) {
	// Create a dummy export request
	exportReq := PrepareDummyExportRequest()

	// Initialize a V2WriteRequestBuilder
	builder, err := NewV2RequestBuilder(exportReq, "http-config")
	if err != nil {
		t.Fatal("unexpected error occurred", err)
	}

	// Build the V2 remote write request
	builder.CreateRequest()

	v2Request := builder.request
	// Verify the generated request
	// Check that the number of TimeSeries is correct
	expectedTimeSeriesCount := 5 // 1 resource, 1 scope, 5 metrics
	assert.Equal(t, expectedTimeSeriesCount, len(v2Request.Timeseries), "The number of TimeSeries should be %v", expectedTimeSeriesCount)

	// Verify that each TimeSeries contains the expected attributes
	for _, ts := range v2Request.Timeseries {
		assert.Equal(t, 1, len(ts.Histograms), "Each TimeSeries should contain exactly one histogram")
		assert.Equal(t, 155, int(ts.Histograms[0].Sum), "The histogram sum should be 155")
		assert.Equal(t, int32(7), ts.Histograms[0].Schema, "The histogram schema should be 7")
	}

	// Check that the symbols table has been populated correctly
	assert.NotEmpty(t, v2Request.Symbols, "The symbols table should not be empty")

    // The following lines of code don't "do" anything
    // Encode the request
    builder.encoder.Encode()
    // Send it to a HTTP URL.
    builder.createHTTPClient()
    builder.send()
}

func TestBuildLabelsUsingLabelRef(t *testing.T) {
	var labelRefs []uint32
	symbolsTable := []string{"", "name", "prometheus"}
	symbols := strings.Join(symbolsTable, "")

	labelNameRef := packSymbol(4, 0)
	labelValueRef := packSymbol(10, 4)
	labelRefs = append(labelRefs, labelNameRef, labelValueRef)

	labels := buildLabelsFromLabelRef(symbols, labelRefs)
	expectedLabels := map[string]string{"name": "prometheus"}
	assert.Equal(t, labels, expectedLabels, "The labels should match the expected name-value pairs.")

}
