// Copyright The OpenTelemetry Authors
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

package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor/internal"
)

const (
	TypeStr          = "ec2"
	cloudProviderAWS = "aws"
)

var _ internal.Detector = (*Detector)(nil)

type Detector struct {
	provider ec2MetadataProvider
}

func NewDetector() (internal.Detector, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return &Detector{provider: &ec2MetadataImpl{sess: sess}}, nil
}

func (d *Detector) Detect(ctx context.Context) (pdata.Resource, error) {
	res := pdata.NewResource()
	res.InitEmpty()

	if !d.provider.available(ctx) {
		return res, nil
	}

	meta, err := d.provider.get(ctx)
	if err != nil {
		return res, err
	}

	attr := res.Attributes()
	attr.InsertString(conventions.AttributeCloudProvider, cloudProviderAWS)
	attr.InsertString(conventions.AttributeCloudRegion, meta.Region)
	attr.InsertString(conventions.AttributeCloudAccount, meta.AccountID)
	attr.InsertString(conventions.AttributeCloudZone, meta.AvailabilityZone)
	attr.InsertString(conventions.AttributeHostID, meta.InstanceID)
	attr.InsertString(conventions.AttributeHostImageID, meta.ImageID)
	attr.InsertString(conventions.AttributeHostType, meta.InstanceType)

	return res, nil
}
