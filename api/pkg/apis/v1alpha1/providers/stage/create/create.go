/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package create

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/providers/metrics"
	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/providers/stage"
	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/utils"
	api_utils "github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/utils"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/contexts"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/observability"
	observ_utils "github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/observability/utils"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/eclipse-symphony/symphony/coa/pkg/logger"
)

const (
	loggerName   = "providers.stage.create"
	providerName = "P (Create Stage)"
	create       = "create"
)

var (
	msLock                   sync.Mutex
	mLog                     = logger.NewLogger(loggerName)
	once                     sync.Once
	providerOperationMetrics *metrics.Metrics
)

type CreateStageProviderConfig struct {
	User         string `json:"user"`
	Password     string `json:"password"`
	WaitCount    int    `json:"wait.count,omitempty"`
	WaitInterval int    `json:"wait.interval,omitempty"`
}

type CreateStageProvider struct {
	Config    CreateStageProviderConfig
	Context   *contexts.ManagerContext
	ApiClient api_utils.ApiClient
}

const (
	RemoveAction = "remove"
	CreateAction = "create"
)

func (s *CreateStageProvider) Init(config providers.IProviderConfig) error {
	ctx, span := observability.StartSpan("[Stage] Create Provider", context.TODO(), &map[string]string{
		"method": "Init",
	})
	var err error = nil
	defer observ_utils.CloseSpanWithError(span, &err)
	defer observ_utils.EmitUserDiagnosticsLogs(ctx, &err)
	msLock.Lock()
	defer msLock.Unlock()
	var mockConfig CreateStageProviderConfig
	mockConfig, err = toSymphonyStageProviderConfig(config)
	if err != nil {
		return err
	}
	s.Config = mockConfig
	s.ApiClient, err = api_utils.GetApiClient()
	if err != nil {
		return err
	}
	once.Do(func() {
		if providerOperationMetrics == nil {
			providerOperationMetrics, err = metrics.New()
			if err != nil {
				mLog.ErrorfCtx(ctx, "  P (Create Stage): failed to create metrics: %+v", err)
			}
		}
	})
	return err
}
func (s *CreateStageProvider) SetContext(ctx *contexts.ManagerContext) {
	s.Context = ctx
}
func toSymphonyStageProviderConfig(config providers.IProviderConfig) (CreateStageProviderConfig, error) {
	ret := CreateStageProviderConfig{}
	data, err := json.Marshal(config)
	if err != nil {
		return ret, err
	}
	err = json.Unmarshal(data, &ret)
	return ret, err
}
func (i *CreateStageProvider) InitWithMap(properties map[string]string) error {
	config, err := SymphonyStageProviderConfigFromMap(properties)
	if err != nil {
		return err
	}
	return i.Init(config)
}
func SymphonyStageProviderConfigFromMap(properties map[string]string) (CreateStageProviderConfig, error) {
	ret := CreateStageProviderConfig{}
	if api_utils.ShouldUseUserCreds() {
		user, err := api_utils.GetString(properties, "user")
		if err != nil {
			return ret, err
		}
		ret.User = user
		if ret.User == "" && !api_utils.ShouldUseSATokens() {
			return ret, v1alpha2.NewCOAError(nil, "user is required", v1alpha2.BadConfig)
		}
		password, err := api_utils.GetString(properties, "password")
		ret.Password = password
		if err != nil {
			return ret, err
		}
	}
	waitStr, err := api_utils.GetString(properties, "wait.count")
	if err != nil {
		return ret, err
	}
	waitCount, err := strconv.Atoi(waitStr)
	if err != nil {
		return ret, v1alpha2.NewCOAError(err, "wait.count must be an integer", v1alpha2.BadConfig)
	}
	ret.WaitCount = waitCount
	waitStr, err = api_utils.GetString(properties, "wait.interval")
	if err != nil {
		return ret, err
	}
	waitInterval, err := strconv.Atoi(waitStr)
	if err != nil {
		return ret, v1alpha2.NewCOAError(err, "wait.interval must be an integer", v1alpha2.BadConfig)
	}
	ret.WaitInterval = waitInterval
	if waitCount <= 0 {
		waitCount = 1
	}
	return ret, nil
}
func (i *CreateStageProvider) Process(ctx context.Context, mgrContext contexts.ManagerContext, inputs map[string]interface{}) (map[string]interface{}, bool, error) {
	ctx, span := observability.StartSpan("[Stage] Create provider", ctx, &map[string]string{
		"method": "Process",
	})
	var err error = nil
	defer observ_utils.CloseSpanWithError(span, &err)
	defer observ_utils.EmitUserDiagnosticsLogs(ctx, &err)

	mLog.InfofCtx(ctx, "  P (Create Stage) process started")
	processTime := time.Now().UTC()
	functionName := observ_utils.GetFunctionName()
	defer providerOperationMetrics.ProviderOperationLatency(
		processTime,
		create,
		metrics.ProcessOperation,
		metrics.RunOperationType,
		functionName,
	)

	outputs := make(map[string]interface{})
	objectType := stage.ReadInputString(inputs, "objectType")
	objectName := stage.ReadInputString(inputs, "objectName")
	action := stage.ReadInputString(inputs, "action")
	object := inputs["object"]
	var objectData []byte
	if object != nil {
		objectData, _ = json.Marshal(object)
	}
	objectName = api_utils.ConvertReferenceToObjectName(objectName)
	lastSummaryMessage := ""
	switch objectType {
	case "instance":
		objectNamespace := stage.GetNamespace(inputs)
		if objectNamespace == "" {
			objectNamespace = "default"
		}

		if strings.EqualFold(action, RemoveAction) {
			observ_utils.EmitUserAuditsLogs(ctx, "  P (Create Stage): Start to delete instance name %s namespace %s", objectName, objectNamespace)
			err = i.ApiClient.DeleteInstance(ctx, objectName, objectNamespace, i.Config.User, i.Config.Password)
			if err != nil {
				providerOperationMetrics.ProviderOperationErrors(
					create,
					functionName,
					metrics.ProcessOperation,
					metrics.RunOperationType,
					v1alpha2.DeleteInstanceFailed.String(),
				)
				mLog.ErrorfCtx(ctx, "  P (Create Stage) process failed, failed to delete instance: %+v", err)
				return nil, false, err
			}
			for ic := 0; ic < i.Config.WaitCount; ic++ {
				remainings := utils.FilterIncompleteDelete(ctx, &i.ApiClient, objectNamespace, []string{objectName}, true, i.Config.User, i.Config.Password)
				if len(remainings) == 0 {
					return outputs, false, nil
				}
				mLog.InfofCtx(ctx, "  P (Create Stage) process: Waiting for instance deletion: %+v", remainings)
				time.Sleep(time.Duration(i.Config.WaitInterval) * time.Second)
			}
			providerOperationMetrics.ProviderOperationErrors(
				create,
				functionName,
				metrics.ProcessOperation,
				metrics.RunOperationType,
				v1alpha2.DeploymentNotReached.String(),
			)
			err = v1alpha2.NewCOAError(nil, "Instance deletion reconcile timeout", v1alpha2.InternalError)
			mLog.ErrorfCtx(ctx, "  P (Create Stage) process failed, error: %+v", err)
			return nil, false, err
		} else if strings.EqualFold(action, CreateAction) {
			var instanceState model.InstanceState
			err = json.Unmarshal(objectData, &instanceState)
			if err != nil {
				mLog.ErrorfCtx(ctx, "Failed to unmarshal instance state %s: %s", objectName, err.Error())
				providerOperationMetrics.ProviderOperationErrors(
					create,
					functionName,
					metrics.ProcessOperation,
					metrics.RunOperationType,
					v1alpha2.CreateInstanceFailed.String(),
				)
				return outputs, false, err
			}

			label_key := os.Getenv("LABEL_KEY")
			label_value := os.Getenv("LABEL_VALUE")
			annotation_name := os.Getenv("ANNOTATION_KEY")
			if label_key != "" && label_value != "" {
				// Check if labels exists within metadata, if not create it
				labels := instanceState.ObjectMeta.Labels
				if labels == nil {
					labels = make(map[string]string)
					instanceState.ObjectMeta.Labels = labels
				}
				// Add the label
				labels[label_key] = label_value
			}
			if annotation_name != "" {
				instanceState.ObjectMeta.UpdateAnnotation(annotation_name, objectName)
			}
			instanceState.ObjectMeta.Namespace = objectNamespace
			instanceState.ObjectMeta.Name = objectName

			if instanceState.ObjectMeta.Name == "" {
				mLog.ErrorfCtx(ctx, "Instance name is empty: - %s", objectName)
				providerOperationMetrics.ProviderOperationErrors(
					create,
					functionName,
					metrics.ProcessOperation,
					metrics.RunOperationType,
					v1alpha2.CreateInstanceFailed.String(),
				)
				return outputs, false, v1alpha2.NewCOAError(nil, fmt.Sprintf("Empty instance name: - %s", objectName), v1alpha2.BadRequest)
			}

			objectData, _ := json.Marshal(instanceState)
			observ_utils.EmitUserAuditsLogs(ctx, "  P (Create Stage): Start to create instance name %s namespace %s", objectName, objectNamespace)
			err = i.ApiClient.CreateInstance(ctx, objectName, objectData, objectNamespace, i.Config.User, i.Config.Password)
			if err != nil {
				providerOperationMetrics.ProviderOperationErrors(
					create,
					functionName,
					metrics.ProcessOperation,
					metrics.RunOperationType,
					v1alpha2.CreateInstanceFailed.String(),
				)
				mLog.ErrorfCtx(ctx, "  P (Create Stage) process failed, failed to create instance: %+v", err)
				return nil, false, err
			}
			for ic := 0; ic < i.Config.WaitCount; ic++ {
				remaining, failed := utils.FilterIncompleteDeploymentUsingSummary(ctx, &i.ApiClient, objectNamespace, []string{objectName}, true, i.Config.User, i.Config.Password)
				if len(remaining) == 0 {
					outputs["objectType"] = objectType
					outputs["objectName"] = objectName
					mLog.InfofCtx(ctx, "  P (Create Stage) process completed with fail count is %d", len(failed))
					if len(failed) > 0 {
						outputs["failedDeployment"] = failed
					}
					return outputs, false, nil
				}
				time.Sleep(time.Duration(i.Config.WaitInterval) * time.Second)
			}
			providerOperationMetrics.ProviderOperationErrors(
				create,
				functionName,
				metrics.ProcessOperation,
				metrics.RunOperationType,
				v1alpha2.DeploymentNotReached.String(),
			)
			err = v1alpha2.NewCOAError(nil, fmt.Sprintf("Instance creation reconcile failed: %s", lastSummaryMessage), v1alpha2.InternalError)
			mLog.ErrorfCtx(ctx, "  P (Create Stage) process failed, error: %+v", err)
			return nil, false, err
		} else {
			err = v1alpha2.NewCOAError(nil, fmt.Sprintf("Unsupported action: %s", action), v1alpha2.BadRequest)
			providerOperationMetrics.ProviderOperationErrors(
				create,
				functionName,
				metrics.ProcessOperation,
				metrics.RunOperationType,
				v1alpha2.UnsupportedAction.String(),
			)
			mLog.ErrorfCtx(ctx, "  P (Create Stage) process failed, error: %+v", err)
			return nil, false, err
		}
	default:
		err = v1alpha2.NewCOAError(nil, fmt.Sprintf("Unsupported object type: %s", objectType), v1alpha2.BadRequest)
		mLog.ErrorfCtx(ctx, "  P (Create Stage) process failed, error: %+v", err)
		providerOperationMetrics.ProviderOperationErrors(
			create,
			functionName,
			metrics.ProcessOperation,
			metrics.RunOperationType,
			v1alpha2.InvalidObjectType.String(),
		)
		return nil, false, err
	}
}
