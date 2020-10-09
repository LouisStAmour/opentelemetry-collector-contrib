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

import "go.opentelemetry.io/collector/consumer/pdata"

/*
	The remainder of these methods are for building up test assets
*/
func getDefaultMetric() pdata.Metric {
	tv := pdata.NewMetric()
	tv.InitEmpty()
	tv.SetName("test_name")
	tv.SetDescription("test_description")
	tv.SetUnit("1")
	tv.IntGauge().InitEmpty()
	tv.IntGauge().DataPoints().Resize(7)
	for i := 0; i < tv.IntGauge().DataPoints().Len(); i++ {
		dp := tv.IntGauge().DataPoints().At(i)
		dp.LabelsMap().InitFromMap(map[string]string{
			"k": "v",
		})
		dp.SetStartTime(pdata.TimestampUnixNano(1234567890))
		dp.SetTimestamp(pdata.TimestampUnixNano(1234567890))
		dp.SetValue(int64(-17))
		dp.Exemplars().Resize(7)
		for i := 0; i < dp.Exemplars().Len(); i++ {
			ex := dp.Exemplars().At(i)
			ex.SetTimestamp(pdata.TimestampUnixNano(1234567890))
			ex.SetValue(int64(-17))
			ex.FilteredLabels().InitFromMap(map[string]string{
				"k": "v",
			})
		}
	}
	return tv
}
