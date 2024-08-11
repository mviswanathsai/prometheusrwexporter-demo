# Prometheus Remote Write v2 Exporter (PoC)

This project is a Proof of Concept (PoC) that showcases the process of building label references and creating a Prometheus Remote Write v2 request from an OTLP export request. The primary goal is to demonstrate a clear understanding of the mechanics involved in this process, rather than focusing on the complete conversion from OTLP to Prometheus Remote Write.

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [Assumptions](#assumptions)
- [Installation](#installation)
- [Usage](#usage)
- [Project Structure](#project-structure)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Overview

This project provides a simplified implementation to illustrate the steps required to transform OpenTelemetry metrics into Prometheus Remote Write v2 requests. It emphasizes the process of building label references (labelRefs) and assembling the Prometheus remotewrite v2 write request.

## Key Concepts

- **Symbol Reference Building**: The process of creating references to metric label value pairs (or, symbols), which are then used to minimize the size of Remote Write requests.
- **V2 Write Request Creation**: Constructing a Prometheus Remote Write v2 request from an OTLP export request, showcasing the flow from OpenTelemetry data to Prometheus-compatible metrics.
- **Proof of Concept**: This implementation is a PoC, intended to demonstrate the understanding of key processes rather than providing a production-ready solution.
- **Encoding**: The specification is still marked experimental, and one of the ways we are still exploring is the use of other encoding libraries. To facilitate development in this direction, an encoder interface has been defined to decouple the builder and the encoder logic.

## Assumptions

- **Omitting Scope Attributes**: For simplicity, scope-related attributes are omitted in the final Prometheus Remote Write (PRW) export.
- **Label Consistency**: It is assumed that all data points for a given metric share the same set of labels, allowing the label reference to be generated from a single data point. i.e, 1 metric = 1 Timeseries.
- **Resource Attributes**: It is assumed that the metrics generated from a given resource share the labels/attributes of the resource.
- **Conversion Logic**: The conversion logic from otlp metrics to Prometheus metrics is quite complex. Since that is not the sole focus of this demo, the conversion logic has been replaced with a simpler (read, spoofed) logic.
- **Encoding Implementation**: Snappy has been specified as the encoder to use in the Spec. Since the discussion about "which encoder to use" is still ongoing, I chose to demostrate the decoupling we might want from the "request builder" and the "encoder" by the defining an encoder interface. However, no logic has yet been implemented.

## Installation

### Prerequisites

- Go 1.18+ installed on your machine.
- A working Go environment (`$GOPATH` set up).

### Clone the Repository

```sh
git clone https://github.com/yourusername/prometheusrwexporter-demo.git
cd prometheusrwexporter-demo
```

### Build the Project

```sh
go build
```

### Run Tests

```sh
go test -v ./...
```

## Usage

### Create a V2 Write Request

To create a v2 write request, first prepare an OTLP export request and pass it to the `NewV2RequestBuilder`:

```go
import (
	"fmt"
	"prometheusrwexporter-demo/prometheusremotewritev2"
)

func main() {
	exportReq := prometheusremotewritev2.PrepareDummyExportRequest()
	builder := prometheusremotewritev2.NewV2RequestBuilder(exportReq)
	v2WriteRequest := builder.CreateV2WriteRequest()

	fmt.Printf("V2 Write Request: %+v\n", v2WriteRequest)
}
```

### Building Label References

The `buildLabelsFromLabelRef` function demonstrates how to create label references (labelRefs) from a given symbols string and label reference array:

```go
func buildLabelsFromLabelRef(symbols string, labelRefs []uint32) []prompb.Label {
	// Implementation to unpack the labelRefs and build the corresponding labels.
	return []prompb.Label{}
}
```

## Testing

Unit tests are provided to ensure the functionality of label reference building and v2 write request creation. To run the tests:

```sh
go test -v ./...
```

Example of a test case for label reference building:

```go
func TestBuildLabelsFromLabelRef(t *testing.T) {
    symbols := "nameviswa"
    nameRef := packSymbol(4, 0)
    valueRef := packSymbol(5, 4)
    labelRefs := []uint32{nameRef, valueRef}

    labels := buildLabelsFromLabelRef(symbols, labelRefs)

    expectedLabels := []prompb.Label{
        {Name: "name", Value: "viswa"},
    }

    assert.Equal(t, expectedLabels, labels)
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.

---
