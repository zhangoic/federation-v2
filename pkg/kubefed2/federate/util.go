/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package federate

import (
	"fmt"
	"strings"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func CrdForAPIResource(apiResource metav1.APIResource, validation *apiextv1b1.CustomResourceValidation) *apiextv1b1.CustomResourceDefinition {
	scope := apiextv1b1.ClusterScoped
	if apiResource.Namespaced {
		scope = apiextv1b1.NamespaceScoped
	}
	return &apiextv1b1.CustomResourceDefinition{
		// Explicitly including TypeMeta will ensure it will be
		// serialized properly to yaml.
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: groupQualifiedName(apiResource),
		},
		Spec: apiextv1b1.CustomResourceDefinitionSpec{
			Group:   apiResource.Group,
			Version: apiResource.Version,
			Scope:   scope,
			Names: apiextv1b1.CustomResourceDefinitionNames{
				Plural: apiResource.Name,
				Kind:   apiResource.Kind,
			},
			Validation: validation,
		},
	}
}

func LookupAPIResource(config *rest.Config, key string) (*metav1.APIResource, error) {
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Error creating discovery client: %v", err)
	}

	// TODO(marun) Allow the targeting of a specific group
	// TODO(marun) Allow the targeting of a specific version

	resourceLists, err := client.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("Error listing api resources: %v", err)
	}

	// TODO(marun) Consider using a caching scheme ala kubectl
	lowerKey := strings.ToLower(key)
	var targetResource *metav1.APIResource
	for _, resourceList := range resourceLists {
		for _, resource := range resourceList.APIResources {
			if lowerKey == resource.Name ||
				lowerKey == resource.SingularName ||
				lowerKey == strings.ToLower(resource.Kind) {

				targetResource = &resource
				break
			}
			for _, shortName := range resource.ShortNames {
				if lowerKey == strings.ToLower(shortName) {
					targetResource = &resource
					break
				}
			}
			if targetResource != nil {
				break
			}
		}
		if targetResource != nil {
			// The list holds the GroupVersion for its list of APIResources
			gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
			if err != nil {
				return nil, fmt.Errorf("Error parsing GroupVersion: %v", err)
			}
			targetResource.Group = gv.Group
			targetResource.Version = gv.Version
			break
		}
	}

	if targetResource != nil {
		return targetResource, nil
	}
	return nil, fmt.Errorf("Unable to find api resource named %q.", key)
}

func resourceKey(apiResource metav1.APIResource) string {
	var group string
	if len(apiResource.Group) == 0 {
		group = "core"
	} else {
		group = apiResource.Group
	}
	var version string
	if len(apiResource.Version) == 0 {
		version = "v1"
	} else {
		version = apiResource.Version
	}
	return fmt.Sprintf("%s.%s/%s", apiResource.Name, group, version)
}

func groupQualifiedName(apiResource metav1.APIResource) string {
	if len(apiResource.Group) == 0 {
		return apiResource.Name
	}
	return fmt.Sprintf("%s.%s", apiResource.Name, apiResource.Group)
}
