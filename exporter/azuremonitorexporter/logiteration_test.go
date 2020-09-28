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

import (
	"testing"

	mock "github.com/stretchr/testify/mock"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type mockLogVisitor struct {
	mock.Mock
}

func (v *mockLogVisitor) visit(resource pdata.Resource, instrumentationLibrary pdata.InstrumentationLibrary, logRecord pdata.LogRecord) (ok bool) {
	args := v.Called(resource, instrumentationLibrary, logRecord)
	return args.Bool(0)
}

// Tests the iteration logic over a pdata.Logs type when no ResourceLogs are provided
func TestLogDataIterationNoResourceLogs(t *testing.T) {
	logs := pdata.NewLogs()

	visitor := getMockLogVisitor(true)

	AcceptLogs(logs, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration logic over a pdata.Logs type when a ResourceLogs is nil
func TestLogDataIterationResourceLogsIsNil(t *testing.T) {
	logs := pdata.NewLogs()
	resourceLogs := pdata.NewResourceLogs()
	logs.ResourceLogs().Append(resourceLogs)

	visitor := getMockLogVisitor(true)

	AcceptLogs(logs, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration logic over a pdata.Logs type when a Resource is nil
func TestLogDataIterationResourceIsNil(t *testing.T) {
	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)

	visitor := getMockLogVisitor(true)

	AcceptLogs(logs, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration logic over a pdata.Logs type when InstrumentationLibraryLogs is nil
func TestLogDataIterationInstrumentationLibraryLogsIsNil(t *testing.T) {
	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	rl := logs.ResourceLogs().At(0)
	r := rl.Resource()
	r.InitEmpty()
	instrumentationLibraryLogs := pdata.NewInstrumentationLibraryLogs()
	rl.InstrumentationLibraryLogs().Append(instrumentationLibraryLogs)

	visitor := getMockLogVisitor(true)

	AcceptLogs(logs, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration logic over a pdata.Logs type when there are no Logs
func TestLogDataIterationNoLogs(t *testing.T) {
	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	rl := logs.ResourceLogs().At(0)
	r := rl.Resource()
	r.InitEmpty()
	rl.InstrumentationLibraryLogs().Resize(1)

	visitor := getMockLogVisitor(true)

	AcceptLogs(logs, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration logic over a pdata.Logs type when the LogRecord is nil
func TestLogDataIterationLogRecordIsNil(t *testing.T) {
	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	rl := logs.ResourceLogs().At(0)
	r := rl.Resource()
	r.InitEmpty()
	rl.InstrumentationLibraryLogs().Resize(1)
	illr := rl.InstrumentationLibraryLogs().At(0)
	logRecord := pdata.NewLogRecord()
	illr.Logs().Append(logRecord)

	visitor := getMockLogVisitor(true)

	AcceptLogs(logs, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 0)
}

// Tests the iteration logic if the visitor returns true
func TestLogDataIterationNoShortCircuit(t *testing.T) {
	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	rl := logs.ResourceLogs().At(0)
	r := rl.Resource()
	r.InitEmpty()
	rl.InstrumentationLibraryLogs().Resize(1)
	illr := rl.InstrumentationLibraryLogs().At(0)
	illr.Logs().Resize(2)

	visitor := getMockLogVisitor(true)

	AcceptLogs(logs, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 2)
}

// Tests the iteration logic short circuit if the visitor returns false
func TestLogDataIterationShortCircuit(t *testing.T) {
	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	rl := logs.ResourceLogs().At(0)
	r := rl.Resource()
	r.InitEmpty()
	rl.InstrumentationLibraryLogs().Resize(1)
	illr := rl.InstrumentationLibraryLogs().At(0)
	illr.Logs().Resize(2)

	visitor := getMockLogVisitor(false)

	AcceptLogs(logs, visitor)

	visitor.AssertNumberOfCalls(t, "visit", 1)
}

func getMockLogVisitor(returns bool) *mockLogVisitor {
	visitor := new(mockLogVisitor)
	visitor.On("visit", mock.Anything, mock.Anything, mock.Anything).Return(returns)
	return visitor
}
