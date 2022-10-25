package deploystack

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/api/cloudresourcemanager/v1"
)

var cloudResourceManagerService *cloudresourcemanager.Service

func getCloudResourceManagerService() (*cloudresourcemanager.Service, error) {
	if cloudResourceManagerService != nil {
		return cloudResourceManagerService, nil
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, opts)
	if err != nil {
		return nil, err
	}

	cloudResourceManagerService = svc

	return svc, nil
}

// ProjectNumber will get the project_number for the input projectid
func ProjectNumber(id string) (string, error) {
	resp := ""
	svc, err := getCloudResourceManagerService()
	if err != nil {
		return resp, err
	}

	results, err := svc.Projects.Get(id).Do()
	if err != nil {
		return resp, err
	}

	resp = strconv.Itoa(int(results.ProjectNumber))

	return resp, nil
}

// ListProjects gets a list of the ListProjects a user has access to
func ListProjects() ([]ProjectWithBilling, error) {
	resp := []ProjectWithBilling{}

	svc, err := getCloudResourceManagerService()
	if err != nil {
		return resp, err
	}

	results, err := svc.Projects.List().Filter("lifecycleState=ACTIVE").Do()
	if err != nil {
		return resp, err
	}

	pwb, err := getBillingForProjects(results.Projects)
	if err != nil {
		return resp, err
	}

	sort.Slice(pwb, func(i, j int) bool {
		return strings.ToLower(pwb[i].Name) < strings.ToLower(pwb[j].Name)
	})

	return pwb, nil
}

// ProjectWithBilling is a project with it's billing status
type ProjectWithBilling struct {
	Name           string
	ID             string
	BillingEnabled bool
}

// ToLabledValue converts a ProjectWithBilling to a LabeledValue
func (p ProjectWithBilling) ToLabledValue() LabeledValue {
	r := LabeledValue{Label: p.Name, Value: p.ID}

	if p.BillingEnabled {
		r.Label = fmt.Sprintf("%s%s (Billing Enabled)%s", TERMGREY, p.Name, TERMCLEAR)
	}

	return r
}

// CreateProject does the work of actually creating a new project in your
// GCP account
func CreateProject(project string) error {
	svc, err := getCloudResourceManagerService()
	if err != nil {
		return err
	}

	proj := cloudresourcemanager.Project{Name: project, ProjectId: project}

	_, err = svc.Projects.Create(&proj).Do()
	if err != nil {
		if strings.Contains(err.Error(), "project_id must be at most 30 characters long") {
			return ErrorProjectCreateTooLong
		}
		if strings.Contains(err.Error(), "project_id contains invalid characters") {
			return ErrorProjectInvalidCharacters
		}
		if strings.Contains(err.Error(), "requested entity already exists") {
			return ErrorProjectAlreadyExists
		}

		return err
	}

	return nil
}

// DeleteProject does the work of actually deleting an existing project in
// your GCP account
func DeleteProject(project string) error {
	svc, err := getCloudResourceManagerService()
	if err != nil {
		return err
	}

	_, err = svc.Projects.Delete(project).Do()
	if err != nil {
		return err
	}

	return nil
}
