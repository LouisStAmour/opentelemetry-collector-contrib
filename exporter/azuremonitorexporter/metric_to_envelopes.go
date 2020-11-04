// Copyright OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azuremonitorexporter

// Contains code common to both trace and metrics exporters
import (
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	"go.uber.org/zap"
	"strconv"
	"time"
)

// Transforms a tuple of pdata.Resource, pdata.InstrumentationLibrary, pdata.LogRecord into an AppInsights contracts.Envelope
// This is the only method that should be targeted in the unit tests
func metricToEnvelopes(
	resource pdata.Resource,
	instrumentationLibrary pdata.InstrumentationLibrary,
	metric pdata.Metric,
	logger *zap.Logger) ([]*contracts.Envelope, error) {

	dropped := 0

	if metric.IsNil() {
		// TODO: Do something with "dropped"
		// TODO: Do not return nil, throw an error or otherwise catch the dropped metric.
		dropped++
		return nil, nil
	}

	var envelopes []*contracts.Envelope

	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		intGauge := metric.IntGauge()
		if intGauge.IsNil() {
			dropped++
			return nil, nil
		}
		dps := intGauge.DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			if dp.IsNil() {
				dropped++
				continue
			}
			envelope := contracts.NewEnvelope()
			envelope.Tags = make(map[string]string)
			envelope.Time = toTime(dp.Timestamp()).Format(time.RFC3339Nano)
			data := contracts.NewMetricData()
			dp.LabelsMap().ForEach( func(k string, v string) {
				data.Properties[k] = v
			})
			dataPoint := contracts.NewDataPoint()
			dataPoint.Name = metric.Name()
			dataPoint.Value = float64(dp.Value())
			data.Metrics = []*contracts.DataPoint { dataPoint }
			envelope.Data = data

			envelopes = append(envelopes, envelope)
		}
	case pdata.MetricDataTypeDoubleGauge:
		doubleGauge := metric.DoubleGauge()
		if doubleGauge.IsNil() {
			dropped++
			return nil, nil
		}
		dps := doubleGauge.DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			if dp.IsNil() {
				dropped++
				continue
			}
			envelope := contracts.NewEnvelope()
			envelope.Tags = make(map[string]string)
			envelope.Time = toTime(dp.Timestamp()).Format(time.RFC3339Nano)
			data := contracts.NewMetricData()
			dp.LabelsMap().ForEach( func(k string, v string) {
				data.Properties[k] = v
			})
			dataPoint := contracts.NewDataPoint()
			dataPoint.Name = metric.Name()
			dataPoint.Value = dp.Value()
			data.Metrics = []*contracts.DataPoint { dataPoint }
			envelope.Data = data

			envelopes = append(envelopes, envelope)
		}
	case pdata.MetricDataTypeIntSum:
		intSum := metric.IntSum()
		if intSum.IsNil() {
			dropped++
			return nil, nil
		}
		dps := intSum.DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			if dp.IsNil() {
				dropped++
				continue
			}
			envelope := contracts.NewEnvelope()
			envelope.Tags = make(map[string]string)
			envelope.Time = toTime(dp.Timestamp()).Format(time.RFC3339Nano)
			data := contracts.NewMetricData()
			dp.LabelsMap().ForEach( func(k string, v string) {
				data.Properties[k] = v
			})
			dataPoint := contracts.NewDataPoint()
			dataPoint.Name = metric.Name()
			dataPoint.Value = float64(dp.Value())
			data.Metrics = []*contracts.DataPoint { dataPoint }
			envelope.Data = data

			envelopes = append(envelopes, envelope)
		}
	case pdata.MetricDataTypeDoubleSum:
		doubleSum := metric.DoubleSum()
		if doubleSum.IsNil() {
			dropped++
			return nil, nil
		}
		dps := doubleSum.DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			if dp.IsNil() {
				dropped++
				continue
			}
			envelope := contracts.NewEnvelope()
			envelope.Tags = make(map[string]string)
			envelope.Time = toTime(dp.Timestamp()).Format(time.RFC3339Nano)
			data := contracts.NewMetricData()
			dp.LabelsMap().ForEach( func(k string, v string) {
				data.Properties[k] = v
			})
			dataPoint := contracts.NewDataPoint()
			dataPoint.Name = metric.Name()
			dataPoint.Value = dp.Value()
			data.Metrics = []*contracts.DataPoint { dataPoint }
			envelope.Data = data

			envelopes = append(envelopes, envelope)
		}
	case pdata.MetricDataTypeIntHistogram:
		intHistogram := metric.IntHistogram()
		if intHistogram.IsNil() {
			dropped++
			return nil, nil
		}
		dps := intHistogram.DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			if dp.IsNil() {
				dropped++
				continue
			}
			envelopeCount := contracts.NewEnvelope()
			envelopeCount.Tags = make(map[string]string)
			envelopeCount.Time = toTime(dp.Timestamp()).Format(time.RFC3339Nano)
			data := contracts.NewMetricData()
			dp.LabelsMap().ForEach( func(k string, v string) {
				data.Properties[k] = v
			})
			dataPoint := contracts.NewDataPoint()
			dataPoint.Name = metric.Name() + "_count"
			dataPoint.Value = float64(dp.Count())
			data.Metrics = []*contracts.DataPoint { dataPoint }
			envelopeCount.Data = data

			envelope := contracts.NewEnvelope()
			envelope.Tags = make(map[string]string)
			envelope.Time = toTime(dp.Timestamp()).Format(time.RFC3339Nano)
			data = contracts.NewMetricData()
			dp.LabelsMap().ForEach( func(k string, v string) {
				data.Properties[k] = v
			})
			dataPoint = contracts.NewDataPoint()
			dataPoint.Name = metric.Name()
			dataPoint.Value = float64(dp.Sum())
			data.Metrics = []*contracts.DataPoint { dataPoint }
			envelope.Data = data

			envelopes = append(envelopes, envelope, envelopeCount)

			bucketName := metric.Name() + "_bucket"
			for j := 0; j < len(dp.BucketCounts()); j++ {
				envelopeBucket := contracts.NewEnvelope()
				envelopeBucket.Tags = make(map[string]string)
				envelopeBucket.Time = envelope.Time
				md := contracts.NewMetricData()
				dp.LabelsMap().ForEach( func(k string, v string) {
					md.Properties[k] = v
				})
				if j < len(dp.ExplicitBounds()) {
					md.Properties["upper_bound"] = strconv.FormatFloat(dp.ExplicitBounds()[j], 'f', -1, 64)
				}
				dataPoint := contracts.NewDataPoint()
				dataPoint.Name = bucketName
				dataPoint.Value = float64(dp.BucketCounts()[j])
				md.Metrics = []*contracts.DataPoint { dataPoint }
				envelopeBucket.Data = md
				envelopes = append(envelopes, envelopeBucket)
			}
		}
	case pdata.MetricDataTypeDoubleHistogram:
		doubleHistogram := metric.DoubleHistogram()
		if doubleHistogram.IsNil() {
			dropped++
			return nil, nil
		}
		dps := doubleHistogram.DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			if dp.IsNil() {
				dropped++
				continue
			}
			envelopeCount := contracts.NewEnvelope()
			envelopeCount.Tags = make(map[string]string)
			envelopeCount.Time = toTime(dp.Timestamp()).Format(time.RFC3339Nano)
			data := contracts.NewMetricData()
			dp.LabelsMap().ForEach( func(k string, v string) {
				data.Properties[k] = v
			})
			dataPoint := contracts.NewDataPoint()
			dataPoint.Name = metric.Name() + "_count"
			dataPoint.Value = float64(dp.Count())
			data.Metrics = []*contracts.DataPoint { dataPoint }
			envelopeCount.Data = data

			envelope := contracts.NewEnvelope()
			envelope.Tags = make(map[string]string)
			envelope.Time = toTime(dp.Timestamp()).Format(time.RFC3339Nano)
			data = contracts.NewMetricData()
			dp.LabelsMap().ForEach( func(k string, v string) {
				data.Properties[k] = v
			})
			dataPoint = contracts.NewDataPoint()
			dataPoint.Name = metric.Name()
			dataPoint.Value = dp.Sum()
			data.Metrics = []*contracts.DataPoint { dataPoint }
			envelope.Data = data

			envelopes = append(envelopes, envelope, envelopeCount)

			bucketName := metric.Name() + "_bucket"
			for j := 0; j < len(dp.BucketCounts()); j++ {
				envelopeBucket := contracts.NewEnvelope()
				envelopeBucket.Tags = make(map[string]string)
				envelopeBucket.Time = envelope.Time
				md := contracts.NewMetricData()
				if j < len(dp.ExplicitBounds()) {
					md.Properties["upper_bound"] = strconv.FormatFloat(dp.ExplicitBounds()[j], 'f', -1, 64)
				}
				dataPoint := contracts.NewDataPoint()
				dataPoint.Name = bucketName
				dataPoint.Value = float64(dp.BucketCounts()[j])
				md.Metrics = []*contracts.DataPoint { dataPoint }
				envelopeBucket.Data = md
				envelopes = append(envelopes, envelopeBucket)
			}
		}
	default:
		// Unknown type, so just increment dropped by 1 as a best effort.
		dropped++
	}

	resourceAttributes := resource.Attributes()

	for i := 0; i < len(envelopes); i++ {
		envelope := envelopes[i]
		data := envelope.Data.(contracts.MetricData)

		// Copy all the resource labels into the base data properties. Resource values are always strings
		resourceAttributes.ForEach(func(k string, v pdata.AttributeValue) { data.Properties[k] = v.StringVal() })

		// Copy the instrumentation properties
		if !instrumentationLibrary.IsNil() {
			if instrumentationLibrary.Name() != "" {
				data.Properties[instrumentationLibraryName] = instrumentationLibrary.Name()
			}

			if instrumentationLibrary.Version() != "" {
				data.Properties[instrumentationLibraryVersion] = instrumentationLibrary.Version()
			}
		}

		// Extract key service.* labels from the Resource labels and construct CloudRole and CloudRoleInstance envelope tags
		// https://github.com/open-telemetry/opentelemetry-specification/tree/master/specification/resource/semantic_conventions
		if serviceName, serviceNameExists := resourceAttributes.Get(conventions.AttributeServiceName); serviceNameExists {
			cloudRole := serviceName.StringVal()

			if serviceNamespace, serviceNamespaceExists := resourceAttributes.Get(conventions.AttributeServiceNamespace); serviceNamespaceExists {
				cloudRole = serviceNamespace.StringVal() + "." + cloudRole
			}

			envelope.Tags[contracts.CloudRole] = cloudRole
		}

		if serviceInstance, exists := resourceAttributes.Get(conventions.AttributeServiceInstance); exists {
			envelope.Tags[contracts.CloudRoleInstance] = serviceInstance.StringVal()
		}

		// Sanitize the base data, the envelope and envelope tags
		sanitize(func() []string { return data.Sanitize() }, logger)
		sanitize(func() []string { return envelope.Sanitize() }, logger)
		sanitize(func() []string { return contracts.SanitizeTags(envelope.Tags) }, logger)
	}

	return envelopes, nil
}