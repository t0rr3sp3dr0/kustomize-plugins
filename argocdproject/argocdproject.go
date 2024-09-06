package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application"
	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	separatorPanic = ": "
	separatorYaml  = "---\n"

	stagingEnvironment = "staging"

	yamlStatusField = "status"
)

type accessLevel int

const (
	ReadOnly accessLevel = iota
	ReadSync
)

func (a accessLevel) String() string {
	switch a {
	case ReadOnly:
		return "read-only"
	case ReadSync:
		return "read-sync"
	default:
		panic(fmt.Sprintf("unknown access level %d", a))
	}
}

func (a accessLevel) Policies(appProjectName string, environment string) []string {
	switch a {
	case ReadOnly:
		return []string{
			fmt.Sprintf("p, proj:%[1]s:%[2]s, *, get, %[1]s/*, allow", appProjectName, ReadOnly),
		}

	case ReadSync:
		defaultPolicies := []string{
			fmt.Sprintf("p, proj:%[1]s:%[2]s, applications, action/apps/Deployment/restart, %[1]s/*, allow", appProjectName, ReadSync),
			fmt.Sprintf("p, proj:%[1]s:%[2]s, applications, action/argoproj.io/Rollout/abort, %[1]s/*, allow", appProjectName, ReadSync),
			fmt.Sprintf("p, proj:%[1]s:%[2]s, applications, action/argoproj.io/Rollout/promote-full, %[1]s/*, allow", appProjectName, ReadSync),
			fmt.Sprintf("p, proj:%[1]s:%[2]s, applications, action/argoproj.io/Rollout/restart, %[1]s/*, allow", appProjectName, ReadSync),
			fmt.Sprintf("p, proj:%[1]s:%[2]s, applications, action/argoproj.io/Rollout/resume, %[1]s/*, allow", appProjectName, ReadSync),
			fmt.Sprintf("p, proj:%[1]s:%[2]s, applications, action/argoproj.io/Rollout/retry, %[1]s/*, allow", appProjectName, ReadSync),
			fmt.Sprintf("p, proj:%[1]s:%[2]s, applications, sync, %[1]s/*, allow", appProjectName, ReadSync),
			fmt.Sprintf("g, proj:%[1]s:%[2]s, proj:%[1]s:%[3]s", appProjectName, ReadSync, ReadOnly),
		}
		if environment == stagingEnvironment {
			defaultPolicies = append(defaultPolicies, fmt.Sprintf("p, proj:%[1]s:%[2]s, applications, override, %[1]s/*, allow", appProjectName, ReadSync))
		}
		return defaultPolicies

	default:
		panic(fmt.Sprintf("unknown access level %d", a))
	}
}

type ArgoCDProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProjectSpec `json:"spec,omitempty"`
}

type ProjectSpec struct {
	AccessControl        AppProjectAccessControl    `json:"accessControl,omitempty"`
	Environment          string                     `json:"environment,omitempty"`
	AppProject           argov1alpha1.AppProject    `json:"appProjectTemplate,omitempty"`
	ApplicationTemplates []argov1alpha1.Application `json:"applicationTemplates,omitempty"`
}

type AppProjectAccessControl struct {
	ReadOnly []string `json:"ReadOnly,omitempty"`
	ReadSync []string `json:"ReadSync,omitempty"`
}

func main() {
	filePath := os.Args[1]

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Panic(filePath, separatorPanic, err)
	}

	if err := GenerateManifests(data, os.Stdout); err != nil {
		log.Panic(filePath, separatorPanic, err)
	}
}

func GenerateManifests(data []byte, out io.Writer) error {
	var argocdProject ArgoCDProject
	if err := yaml.Unmarshal(data, &argocdProject); err != nil {
		return err
	}

	manifests, err := makeManifests(&argocdProject)
	if err != nil {
		return err
	}

	for _, manifest := range manifests {
		if _, err := out.Write([]byte(separatorYaml)); err != nil {
			return err
		}

		if _, err := out.Write(manifest); err != nil {
			return err
		}
	}

	return nil
}

func makeManifests(argocdProject *ArgoCDProject) ([][]byte, error) {
	var manifests [][]byte

	b, err := makeAppProject(argocdProject)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, b)

	bs, err := makeApplications(argocdProject)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, bs...)

	return manifests, nil
}

func makeAppProject(argocdProject *ArgoCDProject) ([]byte, error) {
	appProject := &argocdProject.Spec.AppProject

	appProject.TypeMeta = metav1.TypeMeta{
		APIVersion: argov1alpha1.SchemeGroupVersion.String(),
		Kind:       application.AppProjectKind,
	}

	appProject.Name = argocdProject.Name

	appProject.Spec.NamespaceResourceWhitelist = []metav1.GroupKind{
		metav1.GroupKind{
			Group: "*",
			Kind:  "*",
		},
	}

	// TODO only allow SourceRepos required by applications to avoid unnecessary permissions
	appProject.Spec.SourceRepos = []string{
		"*",
	}

	if appProject.Spec.Destinations == nil {
		destinationMap := make(map[string]argov1alpha1.ApplicationDestination)
		for _, app := range argocdProject.Spec.ApplicationTemplates {
			destinationMap[app.Spec.Destination.String()] = app.Spec.Destination
		}

		destinations := make([]argov1alpha1.ApplicationDestination, 0, len(destinationMap))
		for _, destination := range destinationMap {
			destinations = append(destinations, destination)
		}
		appProject.Spec.Destinations = destinations
	}

	readOnlyProjectRole := makeProjectRole(ReadOnly, argocdProject, appProject)
	appProject.Spec.Roles = append(appProject.Spec.Roles, *readOnlyProjectRole)

	readSyncProjectRole := makeProjectRole(ReadSync, argocdProject, appProject)
	appProject.Spec.Roles = append(appProject.Spec.Roles, *readSyncProjectRole)

	return marshalYAMLWithoutStatusField(appProject)
}

func makeProjectRole(accessLevel accessLevel, argocdProject *ArgoCDProject, appProject *argov1alpha1.AppProject) *argov1alpha1.ProjectRole {
	var groups []string
	switch accessLevel {
	case ReadOnly:
		groups = argocdProject.Spec.AccessControl.ReadOnly
	case ReadSync:
		groups = argocdProject.Spec.AccessControl.ReadSync
	}

	return &argov1alpha1.ProjectRole{
		Name:     accessLevel.String(),
		Policies: accessLevel.Policies(appProject.Name, argocdProject.Spec.Environment),
		Groups:   groups,
	}
}

func makeApplications(argocdProject *ArgoCDProject) ([][]byte, error) {
	apps := argocdProject.Spec.ApplicationTemplates
	manifests := make([][]byte, 0, len(apps))

	for i := range apps {
		app := &apps[i]

		app.TypeMeta = metav1.TypeMeta{
			APIVersion: argov1alpha1.SchemeGroupVersion.String(),
			Kind:       application.ApplicationKind,
		}

		app.Spec.Project = argocdProject.Name

		if argocdProject.Spec.Environment != "" {
			if app.Spec.Source.Path == "" {
				app.Spec.Source.Path = fmt.Sprintf("./k8s/overlays/%s", argocdProject.Spec.Environment)
			}
			app.Spec.Source.TargetRevision = fmt.Sprintf("env-%s", argocdProject.Spec.Environment)
		}

		b, err := marshalYAMLWithoutStatusField(app)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, b)
	}

	return manifests, nil
}

func marshalYAMLWithoutStatusField(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var vm map[string]interface{}
	if err := json.Unmarshal(b, &vm); err != nil {
		return nil, err
	}

	delete(vm, yamlStatusField)

	return yaml.Marshal(vm)
}
