/*

	MIT License

	Copyright (c) Microsoft Corporation.

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE

*/

package create

import (
	"context"
	"os"
	"testing"

	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/contexts"
	"github.com/stretchr/testify/assert"
)

func TestDeployInstance(t *testing.T) {
	testDeploy := os.Getenv("TEST_DEPLOY_INSTANCE")
	if testDeploy != "yes" {
		t.Skip("Skipping becasue TEST_DEPLOY_INSTANCE is missing or not set to 'yes'")
	}
	provider := CreateStageProvider{}
	err := provider.Init(CreateStageProviderConfig{
		BaseUrl:      "http://localhost:8082/v1alpha2/",
		User:         "admin",
		Password:     "",
		WaitCount:    3,
		WaitInterval: 5,
	})
	assert.Nil(t, err)
	outputs, _, err := provider.Process(context.Background(), contexts.ManagerContext{}, map[string]interface{}{
		"objectType": "instance",
		"objectName": "redis-server",
		"object": map[string]interface{}{
			"displayName": "redis-server",
			"solution":    "sample-redis",
			"target": map[string]interface{}{
				"name": "sample-docker-target",
			},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, "OK", outputs["status"])
}
