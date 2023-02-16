package gcloud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/deploystack"
	"google.golang.org/api/compute/v1"
)

// DiskProjects are the list of projects for disk images for Compute Engine
var DiskProjects = deploystack.LabeledValues{
	deploystack.LabeledValue{Label: "CentOS", Value: "centos-cloud"},
	deploystack.LabeledValue{Label: "Container-Optimized OS (COS)", Value: "cos-cloud"},
	// TODO: figure out how to best set this to DefaultImageProject
	deploystack.LabeledValue{Label: "Debian", Value: "debian-cloud", IsDefault: true},
	deploystack.LabeledValue{Label: "Fedora CoreOS", Value: "fedora-coreos-cloud"},
	deploystack.LabeledValue{Label: "Red Hat Enterprise Linux (RHEL)", Value: "rhel-cloud"},
	deploystack.LabeledValue{Label: "Red Hat Enterprise Linux (RHEL) for SAP", Value: "rhel-sap-cloud"},
	deploystack.LabeledValue{Label: "Rocky Linux", Value: "rocky-linux-cloud"},
	deploystack.LabeledValue{Label: "SQL Server", Value: "windows-sql-cloud"},
	deploystack.LabeledValue{Label: "SUSE Linux Enterprise Server (SLES)", Value: "suse-cloud"},
	deploystack.LabeledValue{Label: "SUSE Linux Enterprise Server (SLES) for SAP", Value: "suse-cloud"},
	deploystack.LabeledValue{Label: "SUSE Linux Enterprise Server (SLES) BYOS", Value: "suse-byos-cloud"},
	deploystack.LabeledValue{Label: "Ubuntu LTS", Value: "ubuntu-os-cloud"},
	deploystack.LabeledValue{Label: "Ubuntu Pro", Value: "ubuntu-os-pro-cloud"},
	deploystack.LabeledValue{Label: "Windows Server", Value: "windows-cloud"},
}

func (c *Client) getComputeService(project string) (*compute.Service, error) {
	var err error
	svc := c.services.computeService

	if svc != nil {
		return svc, nil
	}

	if err := c.ServiceEnable(project, "compute.googleapis.com"); err != nil {
		return nil, fmt.Errorf("error activating service for polling: %s", err)
	}

	svc, err = compute.NewService(c.ctx, c.opts)
	if err != nil {
		return nil, err
	}

	svc.UserAgent = c.userAgent
	c.services.computeService = svc

	return svc, nil
}

// ComputeRegionList will return a list of regions for Compute Engine
func (c *Client) ComputeRegionList(project string) ([]string, error) {
	resp := []string{}

	svc, err := c.getComputeService(project)
	if err != nil {
		return resp, err
	}

	results, err := svc.Regions.List(project).Do()
	if err != nil {
		return resp, err
	}

	for _, v := range results.Items {
		resp = append(resp, v.Name)
	}

	sort.Strings(resp)

	return resp, nil
}

// ZoneList will return a list of ComputeZoneList in a given region
func (c *Client) ZoneList(project, region string) ([]string, error) {
	resp := []string{}

	svc, err := c.getComputeService(project)
	if err != nil {
		return resp, err
	}

	filter := fmt.Sprintf("name=%s*", region)

	results, err := svc.Zones.List(project).Filter(filter).Do()
	if err != nil {
		return resp, err
	}

	for _, v := range results.Items {
		resp = append(resp, v.Name)
	}

	sort.Strings(resp)

	return resp, nil
}

// MachineTypeList retrieves the list of Machine Types available in a
// given zone
func (c *Client) MachineTypeList(project, zone string) (*compute.MachineTypeList, error) {
	resp := &compute.MachineTypeList{}

	svc, err := c.getComputeService(project)
	if err != nil {
		return resp, err
	}

	results, err := svc.MachineTypes.List(project, zone).Do()
	if err != nil {
		return resp, err
	}

	return results, nil
}

func formatMBToGB(i int64) string {
	return fmt.Sprintf("%d GB", i/1024)
}

// ImageList gets the list of disk images available for a given image
// project
func (c *Client) ImageList(project, imageproject string) (*compute.ImageList, error) {
	resp := &compute.ImageList{}

	svc, err := c.getComputeService(project)
	if err != nil {
		return resp, err
	}
	results, err := svc.Images.List(imageproject).Do()
	if err != nil {
		return resp, err
	}

	tmp := []*compute.Image{}
	for _, v := range results.Items {
		// fmt.Printf("%v", v.Name)
		if v.Deprecated == nil || v.Deprecated.State == "" {
			// fmt.Printf("- not deprecated")
			tmp = append(tmp, v)
		}

		// fmt.Printf("\n")
	}

	results.Items = tmp

	return results, nil
}

// ImageLatestGet retrieves the latest image from a particular family
func (c *Client) ImageLatestGet(project, imageproject, imagefamily string) (string, error) {
	resp := ""

	svc, err := c.getComputeService(project)
	if err != nil {
		return resp, err
	}

	filter := fmt.Sprintf("(family=\"%s\")", imagefamily)
	results, err := svc.Images.List(imageproject).Filter(filter).Do()
	if err != nil {
		return resp, err
	}

	sort.Slice(results.Items, func(i, j int) bool {
		return results.Items[i].CreationTimestamp > results.Items[j].CreationTimestamp
	})

	for _, v := range results.Items {
		if v.Deprecated == nil || v.Deprecated.State == "" {
			return fmt.Sprintf("%s/%s", imageproject, v.Name), nil
		}
	}

	return "", fmt.Errorf("error: could not find ")
}

// MachineTypeFamilyList gets the list of machine type families
func (c *Client) MachineTypeFamilyList(imgs *compute.MachineTypeList) deploystack.LabeledValues {
	fam := make(map[string]string)
	lb := deploystack.LabeledValues{}

	for _, v := range imgs.Items {
		parts := strings.Split(v.Name, "-")

		key := fmt.Sprintf("%s %s", parts[0], parts[1])
		fam[key] = fmt.Sprintf("%s-%s", parts[0], parts[1])
	}

	for key, value := range fam {
		if key == "" {
			continue
		}
		lb = append(lb, deploystack.LabeledValue{
			Value:     value,
			Label:     key,
			IsDefault: false,
		})
	}
	lb.SetDefault(deploystack.DefaultImageFamily)
	lb.Sort()
	return lb
}

// MachineTypeListByFamily retrieves the list of machine types available
// for each family
func (c *Client) MachineTypeListByFamily(imgs *compute.MachineTypeList, family string) deploystack.LabeledValues {
	lb := deploystack.LabeledValues{}

	tempTypes := []compute.MachineType{}

	for _, v := range imgs.Items {
		if strings.Contains(v.Name, family) {
			tempTypes = append(tempTypes, *v)
		}
	}

	sort.Slice(tempTypes, func(i, j int) bool {
		return tempTypes[i].GuestCpus < tempTypes[j].GuestCpus
	})

	for _, v := range tempTypes {
		if strings.Contains(v.Name, family) {
			value := v.Name
			label := fmt.Sprintf("%s %s", v.Name, v.Description)
			lb = append(lb, deploystack.LabeledValue{
				Value:     value,
				Label:     label,
				IsDefault: false,
			})
		}
	}
	lb.SetDefault(lb[0].Value)

	return lb
}

// ImageFamilyList gets a list of image families
func (c *Client) ImageFamilyList(imgs *compute.ImageList) deploystack.LabeledValues {
	fam := make(map[string]bool)
	lb := deploystack.LabeledValues{}

	for _, v := range imgs.Items {
		fam[v.Family] = false
	}

	for i := range fam {
		if i == "" {
			continue
		}
		lb = append(lb, deploystack.LabeledValue{
			Value:     i,
			Label:     i,
			IsDefault: false,
		})
	}
	lb.SetDefault(deploystack.DefaultImageFamily)
	lb.Sort()
	return lb
}

// ImageTypeListByFamily retrieves a list of iamge types by the family
func (c *Client) ImageTypeListByFamily(imgs *compute.ImageList, project, family string) deploystack.LabeledValues {
	lb := deploystack.LabeledValues{}

	for _, v := range imgs.Items {
		if v.Family == family {
			value := fmt.Sprintf("%s/%s", project, v.Name)
			lb = append(lb, deploystack.LabeledValue{
				Value:     value,
				Label:     v.Name,
				IsDefault: false,
			})
		}
	}

	last := lb[len(lb)-1]
	last.Label = fmt.Sprintf("%s (Latest)", last.Label)
	lb[len(lb)-1] = last
	lb.Sort()
	lb.SetDefault(last.Value)

	return lb
}