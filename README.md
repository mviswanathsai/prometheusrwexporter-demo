# Proof of Concept (PoC) for Prometheus RemoteWrite 2.0 Specification

## Objective

This Proof of Concept (PoC) aims to illustrate the process of converting an OTLP export request into a Prometheus RemoteWrite V2 (RWV2) request, in accordance with the Prometheus RemoteWrite 2.0 Specification. The focus is on the following key aspects:

- Creating a Symbols table.
- Creating LabelReferences.
- Reading from a Symbols table using references.

## Assumptions

1. **Metric Representation:**
   - Each `pmetric.Metric` object represents a single Timeseries. This assumption simplifies the handling of metrics but does not fully reflect real-world scenarios where metrics may have varying attributes (labels). Future iterations of the PoC should address the complexity of handling metrics with different label sets.

2. **Scope Attributes:**
   - The PoC does not account for the InstrumentationScope associated with metrics. This omission is due to the complexity of integrating scope attributes and a lack of clear usage context.

3. **OTLP to Prometheus Metrics Conversion:**
   - Existing Prometheus packages may be leveraged for conversion tasks with minimal adjustments. Histograms are simplified in this PoC due to the time constraints and lack of domain-specific knowledge.

4. **Encoding and Client Behavior:**
   - While Snappy encoding/decoding is outlined in the specification, the PoC separates request building from encoding to maintain clarity. The `encoder` interface is defined but not yet implemented. The HTTP client logic, while important, is excluded from this PoC to focus on core functionality.

## Implementation Highlights

1. **Building a Remote Write V2 Request:**
   - The PoC implements `V2WriteRequestBuilder` to convert OTLP export requests into Prometheus RWV2-compatible write requests. This includes:
     - Constructing a Symbols table for each OTLP export request.
     - Iterating through each Metric to build an array of `ts` objects.
     - Using these `ts` objects to create the final `[]Timeseries` in the RWV2 request.

2. **Symbols Table Creation:**
   - Deduplicating and constructing a Symbols table from metrics and their attributes.
   - Packing symbol length and offset into `uint32`, which is stored in arrays called references.

3. **Reading References:**
   - Interpreting references to generate name-value label pairs from the Symbols table.

4. **Future Enhancements:**
   - Improving error handling with detailed error messages.
   - Refactoring code for better structure and readability.
   - Optimizing implementations based on further iterations and domain-specific knowledge.

## Summary

This PoC provides a foundational demonstration of converting OTLP export requests to Prometheus RemoteWrite V2 requests. It establishes the core functionality, including Symbols table creation, LabelReferences, and data handling. Further refinements and iterations will focus on handling complex metrics, incorporating scope attributes, and optimizing performance.

## Glossary

1. **OTLP (OpenTelemetry Protocol)**
   - **Definition:** OTLP is a standard protocol used by OpenTelemetry to transmit telemetry data, such as traces and metrics, from instrumented applications to back-end systems for monitoring and analysis.
   - **Usage in PoC:** OTLP export requests are converted into Prometheus RemoteWrite V2 requests in the PoC.

2. **Prometheus RemoteWrite V2 (RWV2)**
   - **Definition:** Prometheus RemoteWrite V2 is a specification for sending time-series data from Prometheus to external systems. The V2 version introduces improvements and enhancements over the original RemoteWrite specification, including more efficient encoding and data representation.
   - **Usage in PoC:** The PoC demonstrates how to build and send data in the Prometheus RemoteWrite V2 format from OTLP export requests.


