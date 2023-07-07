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

package docker

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/azure/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/observability"
	observ_utils "github.com/azure/symphony/coa/pkg/apis/v1alpha2/observability/utils"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/azure/symphony/coa/pkg/logger"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var sLog = logger.NewLogger("coa.runtime")

type DockerTargetProviderConfig struct {
	Name string `json:"name"`
}

type DockerTargetProvider struct {
	Config DockerTargetProviderConfig
}

func DockerTargetProviderConfigFromMap(properties map[string]string) (DockerTargetProviderConfig, error) {
	ret := DockerTargetProviderConfig{}
	if v, ok := properties["name"]; ok {
		ret.Name = v
	}
	return ret, nil
}
func (d *DockerTargetProvider) InitWithMap(properties map[string]string) error {
	config, err := DockerTargetProviderConfigFromMap(properties)
	if err != nil {
		return err
	}
	return d.Init(config)
}
func (d *DockerTargetProvider) Init(config providers.IProviderConfig) error {
	_, span := observability.StartSpan("Docker Target Provider", context.Background(), &map[string]string{
		"method": "Init",
	})
	sLog.Info("  P (Docker Target): Init()")

	// convert config to DockerTargetProviderConfig type
	dockerConfig, err := toDockerTargetProviderConfig(config)
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		sLog.Errorf("  P (Docker Target): expected DockerTargetProviderConfig: %+v", err)
		return err
	}

	d.Config = dockerConfig
	return nil
}
func toDockerTargetProviderConfig(config providers.IProviderConfig) (DockerTargetProviderConfig, error) {
	ret := DockerTargetProviderConfig{}
	data, err := json.Marshal(config)
	if err != nil {
		return ret, err
	}
	err = json.Unmarshal(data, &ret)
	return ret, err
}

func (i *DockerTargetProvider) Get(ctx context.Context, deployment model.DeploymentSpec, references []model.ComponentStep) ([]model.ComponentSpec, error) {
	ctx, span := observability.StartSpan("Docker Target Provider", ctx, &map[string]string{
		"method": "Get",
	})
	sLog.Infof("  P (Docker Target): getting artifacts: %s - %s", deployment.Instance.Scope, deployment.Instance.Name)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		sLog.Errorf("  P (Docker Target): failed to create docker client: %+v", err)
		return nil, err
	}

	ret := make([]model.ComponentSpec, 0)
	for _, component := range references {
		info, err := cli.ContainerInspect(ctx, component.Component.Name)
		if err == nil {
			name := info.Name
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
			component := model.ComponentSpec{
				Name:       name,
				Properties: make(map[string]interface{}),
			}
			// container.args
			if len(info.Args) > 0 {
				argsData, _ := json.Marshal(info.Args)
				component.Properties["container.args"] = string(argsData)
			}
			// container.image
			component.Properties[model.ContainerImage] = info.Config.Image
			if info.HostConfig != nil {
				resources, _ := json.Marshal(info.HostConfig.Resources)
				component.Properties["container.resources"] = string(resources)
			}
			// container.ports
			if info.NetworkSettings != nil && len(info.NetworkSettings.Ports) > 0 {
				ports, _ := json.Marshal(info.NetworkSettings.Ports)
				component.Properties["container.ports"] = string(ports)
			}
			// container.cmd
			if len(info.Config.Cmd) > 0 {
				cmdData, _ := json.Marshal(info.Config.Cmd)
				component.Properties["container.commands"] = string(cmdData)
			}
			// container.volumeMounts
			if len(info.Mounts) > 0 {
				volumeData, _ := json.Marshal(info.Mounts)
				component.Properties["container.volumeMounts"] = string(volumeData)
			}
			ret = append(ret, component)
		}
	}

	observ_utils.CloseSpanWithError(span, nil)
	return ret, nil
}

func (i *DockerTargetProvider) Apply(ctx context.Context, deployment model.DeploymentSpec, step model.DeploymentStep, isDryRun bool) (map[string]model.ComponentResultSpec, error) {
	_, span := observability.StartSpan("Docker Target Provider", ctx, &map[string]string{
		"method": "Apply",
	})
	sLog.Infof("  P (Docker Target): applying artifacts: %s - %s", deployment.Instance.Scope, deployment.Instance.Name)

	injections := &model.ValueInjections{
		InstanceId: deployment.Instance.Name,
		SolutionId: deployment.Instance.Solution,
		TargetId:   deployment.ActiveTarget,
	}

	components := step.GetComponents()
	err := i.GetValidationRule(ctx).Validate(components)
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		return nil, err
	}
	if isDryRun {
		observ_utils.CloseSpanWithError(span, nil)
		return nil, nil
	}

	ret := step.PrepareResultMap()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		sLog.Errorf("  P (Docker Target): failed to create docker client: %+v", err)
		return ret, err
	}

	for _, component := range step.Components {
		if component.Action == "update" {
			image := model.ReadPropertyCompat(component.Component.Properties, model.ContainerImage, injections)
			resources := model.ReadPropertyCompat(component.Component.Properties, "container.resources", injections)
			if image == "" {
				err := errors.New("component doesn't have container.image property")
				ret[component.Component.Name] = model.ComponentResultSpec{
					Status:  v1alpha2.UpdateFailed,
					Message: err.Error(),
				}
				observ_utils.CloseSpanWithError(span, err)
				sLog.Errorf("  P (Helm Target): component doesn't have container.image property")
				return ret, err
			}

			isNew := true
			containerInfo, err := cli.ContainerInspect(ctx, component.Component.Name)
			if err == nil {
				isNew = false
			}

			// TODO: I don't think we need to do an explict image pull here, as Docker will pull the image upon cache miss
			// reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
			// if err != nil {
			// 	observ_utils.CloseSpanWithError(span, err)
			// 	sLog.Errorf("  P (Docker Target): failed to pull docker image: %+v", err)
			// 	return err
			// }

			// defer reader.Close()
			// io.Copy(os.Stdout, reader)

			if !isNew && containerInfo.Image != image {
				err = cli.ContainerStop(context.Background(), component.Component.Name, nil)
				if err != nil {
					if !client.IsErrNotFound(err) {
						observ_utils.CloseSpanWithError(span, err)
						sLog.Errorf("  P (Docker Target): failed to stop a running container: %+v", err)
						return ret, err
					}
				}
				err = cli.ContainerRemove(context.Background(), component.Component.Name, types.ContainerRemoveOptions{})
				if err != nil {
					ret[component.Component.Name] = model.ComponentResultSpec{
						Status:  v1alpha2.UpdateFailed,
						Message: err.Error(),
					}
					observ_utils.CloseSpanWithError(span, err)
					sLog.Errorf("  P (Docker Target): failed to remove existing container: %+v", err)
					return ret, err
				}
				isNew = true
			}

			if isNew {
				containerConfig := container.Config{
					Image: image,
				}
				var hostConfig *container.HostConfig
				if resources != "" {
					var resourceSpec container.Resources
					err := json.Unmarshal([]byte(resources), &resourceSpec)
					if err != nil {
						ret[component.Component.Name] = model.ComponentResultSpec{
							Status:  v1alpha2.UpdateFailed,
							Message: err.Error(),
						}
						observ_utils.CloseSpanWithError(span, err)
						sLog.Errorf("  P (Docker Target): failed to read container resource settings: %+v", err)
						return ret, err
					}
					hostConfig = &container.HostConfig{
						Resources: resourceSpec,
					}
				}
				container, err := cli.ContainerCreate(context.Background(), &containerConfig, hostConfig, nil, nil, component.Component.Name)
				if err != nil {
					ret[component.Component.Name] = model.ComponentResultSpec{
						Status:  v1alpha2.UpdateFailed,
						Message: err.Error(),
					}
					observ_utils.CloseSpanWithError(span, err)
					sLog.Errorf("  P (Docker Target): failed to create container: %+v", err)
					return ret, err
				}

				if err := cli.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{}); err != nil {
					ret[component.Component.Name] = model.ComponentResultSpec{
						Status:  v1alpha2.UpdateFailed,
						Message: err.Error(),
					}
					observ_utils.CloseSpanWithError(span, err)
					sLog.Errorf("  P (Docker Target): failed to start container: %+v", err)
					return ret, err
				}
				ret[component.Component.Name] = model.ComponentResultSpec{
					Status:  v1alpha2.Updated,
					Message: "",
				}
			} else {
				if resources != "" {
					var resourceObj container.Resources
					err = json.Unmarshal([]byte(resources), &resourceObj)
					if err != nil {
						ret[component.Component.Name] = model.ComponentResultSpec{
							Status:  v1alpha2.UpdateFailed,
							Message: err.Error(),
						}
						observ_utils.CloseSpanWithError(span, err)
						sLog.Errorf("  P (Docker Target): failed to unmarshal container resources spec: %+v", err)
						return ret, err
					}
					_, err = cli.ContainerUpdate(context.Background(), component.Component.Name, container.UpdateConfig{
						Resources: resourceObj,
					})
					if err != nil {
						ret[component.Component.Name] = model.ComponentResultSpec{
							Status:  v1alpha2.UpdateFailed,
							Message: err.Error(),
						}
						observ_utils.CloseSpanWithError(span, err)
						sLog.Errorf("  P (Docker Target): failed to update container resources: %+v", err)
						return ret, err
					}
				}
				ret[component.Component.Name] = model.ComponentResultSpec{
					Status:  v1alpha2.Updated,
					Message: "",
				}
			}
		} else {
			err = cli.ContainerStop(context.Background(), component.Component.Name, nil)
			if err != nil {
				if !client.IsErrNotFound(err) {
					observ_utils.CloseSpanWithError(span, err)
					sLog.Errorf("  P (Docker Target): failed to stop a running container: %+v", err)
					return ret, err
				}
			}
			err = cli.ContainerRemove(context.Background(), component.Component.Name, types.ContainerRemoveOptions{})
			if err != nil {
				if !client.IsErrNotFound(err) {
					observ_utils.CloseSpanWithError(span, err)
					sLog.Errorf("  P (Docker Target): failed to remove existing container: %+v", err)
					return ret, err
				}
			}
			ret[component.Component.Name] = model.ComponentResultSpec{
				Status:  v1alpha2.Deleted,
				Message: "",
			}
		}
	}
	observ_utils.CloseSpanWithError(span, nil)
	return ret, nil
}

func (*DockerTargetProvider) GetValidationRule(ctx context.Context) model.ValidationRule {
	return model.ValidationRule{
		RequiredProperties:    []string{model.ContainerImage},
		OptionalProperties:    []string{"container.resources"},
		RequiredComponentType: "",
		RequiredMetadata:      []string{},
		OptionalMetadata:      []string{},
		ChangeDetectionProperties: []model.PropertyDesc{
			{Name: model.ContainerImage, IgnoreCase: false, SkipIfMissing: false},
			{Name: "container.ports", IgnoreCase: false, SkipIfMissing: true},
			{Name: "container.resources", IgnoreCase: false, SkipIfMissing: true},
		},
	}
}
