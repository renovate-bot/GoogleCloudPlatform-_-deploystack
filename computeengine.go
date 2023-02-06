package deploystack

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/api/compute/v1"
)

var computeService *compute.Service

// DiskProjects are the list of projects for disk images for Compute Engine
var DiskProjects = LabeledValues{
	LabeledValue{Label: "CentOS", Value: "centos-cloud"},
	LabeledValue{Label: "Container-Optimized OS (COS)", Value: "cos-cloud"},
	// TODO: figure out how to best set this to DefaultImageProject
	LabeledValue{Label: "Debian", Value: "debian-cloud", IsDefault: true},
	LabeledValue{Label: "Fedora CoreOS", Value: "fedora-coreos-cloud"},
	LabeledValue{Label: "Red Hat Enterprise Linux (RHEL)", Value: "rhel-cloud"},
	LabeledValue{Label: "Red Hat Enterprise Linux (RHEL) for SAP", Value: "rhel-sap-cloud"},
	LabeledValue{Label: "Rocky Linux", Value: "rocky-linux-cloud"},
	LabeledValue{Label: "SQL Server", Value: "windows-sql-cloud"},
	LabeledValue{Label: "SUSE Linux Enterprise Server (SLES)", Value: "suse-cloud"},
	LabeledValue{Label: "SUSE Linux Enterprise Server (SLES) for SAP", Value: "suse-cloud"},
	LabeledValue{Label: "SUSE Linux Enterprise Server (SLES) BYOS", Value: "suse-byos-cloud"},
	LabeledValue{Label: "Ubuntu LTS", Value: "ubuntu-os-cloud"},
	LabeledValue{Label: "Ubuntu Pro", Value: "ubuntu-os-pro-cloud"},
	LabeledValue{Label: "Windows Server", Value: "windows-cloud"},
}

// ComputeRegionList will return a list of regions for Compute Engine
func ComputeRegionList(project string) ([]string, error) {
	resp := []string{}

	svc, err := getComputeService(project)
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

func getComputeService(project string) (*compute.Service, error) {
	if computeService != nil {
		return computeService, nil
	}

	if err := ServiceEnable(project, "compute.googleapis.com"); err != nil {
		return nil, fmt.Errorf("error activating service for polling: %s", err)
	}

	ctx := context.Background()
	svc, err := compute.NewService(ctx, opts)
	if err != nil {
		return nil, err
	}

	computeService = svc
	svc.UserAgent = defaultUserAgent

	return svc, nil
}

// ComputeZoneList will return a list of ComputeZoneList in a given region
func ComputeZoneList(project, region string) ([]string, error) {
	resp := []string{}

	svc, err := getComputeService(project)
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

// ComputeMachineTypeList retrieves the list of Machine Types available in a
// given zone
func ComputeMachineTypeList(project, zone string) (*compute.MachineTypeList, error) {
	resp := &compute.MachineTypeList{}

	svc, err := getComputeService(project)
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

// ComputeImageList gets the list of disk images available for a given image
// project
func ComputeImageList(project, imageproject string) (*compute.ImageList, error) {
	resp := &compute.ImageList{}

	svc, err := getComputeService(project)
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

// ComputeImageLatestGet retrieves the latest image from a particular family
func ComputeImageLatestGet(project, imageproject, imagefamily string) (string, error) {
	resp := ""

	svc, err := getComputeService(project)
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

// ComputeMachineTypeFamilyList gets the list of machine type families
func ComputeMachineTypeFamilyList(imgs *compute.MachineTypeList) LabeledValues {
	fam := make(map[string]string)
	lb := LabeledValues{}

	for _, v := range imgs.Items {
		parts := strings.Split(v.Name, "-")

		key := fmt.Sprintf("%s %s", parts[0], parts[1])
		fam[key] = fmt.Sprintf("%s-%s", parts[0], parts[1])
	}

	for key, value := range fam {
		if key == "" {
			continue
		}
		lb = append(lb, LabeledValue{value, key, false})
	}
	lb.SetDefault(DefaultImageFamily)
	lb.Sort()
	return lb
}

// ComputeMachineTypeListByFamily retrieves the list of machine types available
// for each family
func ComputeMachineTypeListByFamily(imgs *compute.MachineTypeList, family string) LabeledValues {
	lb := LabeledValues{}

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
			lb = append(lb, LabeledValue{value, label, false})
		}
	}
	lb.SetDefault(lb[0].Value)

	return lb
}

// ComputeImageFamilyList gets a list of image families
func ComputeImageFamilyList(imgs *compute.ImageList) LabeledValues {
	fam := make(map[string]bool)
	lb := LabeledValues{}

	for _, v := range imgs.Items {
		fam[v.Family] = false
	}

	for i := range fam {
		if i == "" {
			continue
		}
		lb = append(lb, LabeledValue{i, i, false})
	}
	lb.SetDefault(DefaultImageFamily)
	lb.Sort()
	return lb
}

// ComputeImageTypeListByFamily retrieves a list of iamge types by the family
func ComputeImageTypeListByFamily(imgs *compute.ImageList, project, family string) LabeledValues {
	lb := LabeledValues{}

	for _, v := range imgs.Items {
		if v.Family == family {
			value := fmt.Sprintf("%s/%s", project, v.Name)
			lb = append(lb, LabeledValue{value, v.Name, false})
		}
	}

	last := lb[len(lb)-1]
	last.Label = fmt.Sprintf("%s (Latest)", last.Label)
	lb[len(lb)-1] = last
	lb.Sort()
	lb.SetDefault(last.Value)

	return lb
}
