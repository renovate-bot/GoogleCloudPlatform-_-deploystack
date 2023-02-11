// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deploystack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kylelemons/godebug/diff"
	"google.golang.org/api/option"
	domainspb "google.golang.org/genproto/googleapis/cloud/domains/v1beta1"
	"google.golang.org/genproto/googleapis/type/postaladdress"
)

var (
	projectID = ""
	creds     map[string]string
)

func TestMain(m *testing.M) {
	var err error
	opts = option.WithCredentialsFile("creds.json")

	dat, err := os.ReadFile("creds.json")
	if err != nil {
		log.Fatalf("unable to handle the json config file: %v", err)
	}

	json.Unmarshal(dat, &creds)

	projectID = creds["project_id"]
	if err != nil {
		log.Fatalf("could not get environment project id: %s", err)
	}
	code := m.Run()
	os.Exit(code)
}

func TestReadConfig(t *testing.T) {
	errUnableToRead := errors.New("unable to read config file: ")
	tests := map[string]struct {
		path string
		want Stack
		err  error
	}{
		"error": {
			path: "sadasd",
			want: Stack{},
			err:  errUnableToRead,
		},
		"no_custom": {
			path: "test_files/no_customs",
			want: Stack{
				Config: Config{
					Title:         "TESTCONFIG",
					Description:   "A test string for usage with this stuff.",
					Duration:      5,
					Project:       true,
					Region:        true,
					RegionType:    "functions",
					RegionDefault: "us-central1",
				},
			},
			err: nil,
		},
		"custom": {
			path: "test_files/customs",
			want: Stack{
				Config: Config{
					Title:         "TESTCONFIG",
					Description:   "A test string for usage with this stuff.",
					Duration:      5,
					Project:       false,
					Region:        false,
					RegionType:    "",
					RegionDefault: "",
					CustomSettings: []Custom{
						{Name: "nodes", Description: "Nodes", Default: "3"},
					},
				},
			},
			err: nil,
		},
		"custom_options": {
			path: "test_files/customs_options",
			want: Stack{
				Config: Config{
					Title:         "TESTCONFIG",
					Description:   "A test string for usage with this stuff.",
					Duration:      5,
					Project:       false,
					Region:        false,
					RegionType:    "",
					RegionDefault: "",

					CustomSettings: []Custom{
						{
							Name:        "nodes",
							Description: "Nodes",
							Default:     "3",
							Options:     []string{"1", "2", "3"},
						},
					},
				},
			},
			err: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewStack()
			oldWD, _ := os.Getwd()
			os.Chdir(tc.path)

			err := s.FindAndReadRequired()

			if errors.Is(err, tc.err) {
				if err != nil && tc.err != nil && err.Error() != tc.err.Error() {
					t.Fatalf("expected: error(%s) got: error(%s)", tc.err, err)
				}
			}

			os.Chdir(oldWD)

			compareValues("Title", tc.want.Config.Title, s.Config.Title, t)
			compareValues("Description", tc.want.Config.Description, s.Config.Description, t)
			compareValues("Duration", tc.want.Config.Duration, s.Config.Duration, t)
			compareValues("Project", tc.want.Config.Project, s.Config.Project, t)
			compareValues("Region", tc.want.Config.Region, s.Config.Region, t)
			compareValues("RegionType", tc.want.Config.RegionType, s.Config.RegionType, t)
			compareValues("RegionDefault", tc.want.Config.RegionDefault, s.Config.RegionDefault, t)
			for i, v := range s.Config.CustomSettings {
				compareValues(v.Name, tc.want.Config.CustomSettings[i], v, t)
			}
		})
	}
}

func compareValues(label string, want interface{}, got interface{}, t *testing.T) {
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("%s: expected: \n|%v|\ngot: \n|%v|", label, want, got)
	}
}

// func TestProcessCustoms(t *testing.T) {
// 	tests := map[string]struct {
// 		path string
// 		want string
// 		err  error
// 	}{
// 		"custom_options": {
// 			path: "test_files/customs_options",
// 			want: `********************************************************************************[1;36mDeploystack [0m
// Deploystack will walk you through setting some options for the
// stack this solutions installs.
// Most questions have a default that you can choose by hitting the Enter key
// ********************************************************************************[1;36mPress the Enter Key to continue [0m
// ********************************************************************************
// [1;36mTESTCONFIG[0m
// A test string for usage with this stuff.
// It's going to take around [0;36m5 minutes[0m
// ********************************************************************************
// [1;36mNodes: [0m
// 1) 1
// 2) 2
// [1;36m 3) 3 [0m
// Choose number from list, or just [enter] for [1;36m3[0m
// >
// [46mProject Details [0m
// Stack Name: [1;36mtest[0m
// Nodes:      [1;36m3[0m
// `,
// 			err: nil,
// 		},
// 		"custom": {
// 			path: "test_files/customs",
// 			want: `********************************************************************************[1;36mDeploystack [0m
// Deploystack will walk you through setting some options for the
// stack this solutions installs.
// Most questions have a default that you can choose by hitting the Enter key
// ********************************************************************************[1;36mPress the Enter Key to continue [0m
// ********************************************************************************
// [1;36mTESTCONFIG[0m
// A test string for usage with this stuff.
// It's going to take around [0;36m5 minutes[0m
// ********************************************************************************
// [1;36mNodes: [0m
// Enter value, or just [enter] for [1;36m3[0m
// >
// [46mProject Details [0m
// Stack Name: [1;36mtest[0m
// Nodes:      [1;36m3[0m
// `,
// 			err: nil,
// 		},
// 	}

// 	for name, tc := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			s := NewStack()
// 			os.Chdir(tc.path)
// 			err := s.FindAndReadRequired()
// 			if !errors.Is(err, tc.err) {
// 				if err != nil && tc.err != nil && err.Error() != tc.err.Error() {
// 					t.Fatalf("expected: error(%s) got: error(%s)", tc.err, err)
// 				}
// 			}
// 			got := captureOutput(func() {
// 				if err := s.Process("terraform.tfvars"); err != nil {
// 					t.Fatalf("problem collecting the configurations: %s", err)
// 				}
// 			})

// 			if !reflect.DeepEqual(tc.want, got) {
// 				fmt.Println(diff.Diff(got, tc.want))
// 				t.Fatalf("expected: \n|%v|\ngot: \n|%v|", tc.want, got)
// 			}
// 		})
// 	}
// }

func TestStackTFvars(t *testing.T) {
	s := NewStack()
	s.AddSetting("project", "testproject")
	s.AddSetting("boolean", "true")
	s.AddSetting("set", "[item1,item2]")
	got := s.Terraform()

	want := `boolean="true"
project="testproject"
set=["item1","item2"]
`

	if got != want {
		fmt.Println(diff.Diff(want, got))
		t.Fatalf("expected: %v, got: %v", want, got)
	}
}

func TestStackTFvarsWithProjectNAme(t *testing.T) {
	s := NewStack()
	s.AddSetting("project", "testproject")
	s.AddSetting("boolean", "true")
	s.AddSetting("project_name", "dontshow")
	s.AddSetting("set", "[item1,item2]")
	got := s.Terraform()

	want := `boolean="true"
project="testproject"
set=["item1","item2"]
`

	if got != want {
		fmt.Println(diff.Diff(want, got))
		t.Fatalf("expected: %v, got: %v", want, got)
	}
}

func captureOutput(f func()) string {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	return string(out)
}

func blockOutput() (string, *os.File) {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	w.Close()
	out, _ := ioutil.ReadAll(r)
	return string(out), rescueStdout
}

func randSeq(n int) string {
	rand.Seed(time.Now().Unix())

	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func remove(l []string, item string) []string {
	for i, other := range l {
		if other == item {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}

func TestMassgePhoneNumber(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
		err   error
	}{
		"Good":  {"800 555 1234", "+1.8005551234", nil},
		"Weird": {"d746fd83843", "+1.74683843", nil},
		"BAD":   {"dghdhdfuejfhfhfhrghfhfhdhgreh", "", ErrorCustomNotValidPhoneNumber},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := massagePhoneNumber(tc.input)
			if err != tc.err {
				t.Fatalf("expected: %v, got: %v", tc.err, err)
			}
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestCustomCollect(t *testing.T) {
	_, rescueStdout := blockOutput()
	defer func() { os.Stdout = rescueStdout }()
	tests := map[string]struct {
		input  string
		custom Custom
		want   string
	}{
		"UserEntry":      {"test_input", Custom{Name: "test", Default: "broken_test"}, "test_input"},
		"Default":        {"", Custom{Name: "test", Default: "working_test"}, "working_test"},
		"Phone":          {"215-555-5321", Custom{Name: "test", Default: "215-555-5321", Validation: "phonenumber"}, "+1.2155555321"},
		"PhoneDefault":   {"", Custom{Name: "test", Default: "215-555-5321", Validation: "phonenumber"}, "+1.2155555321"},
		"Integer":        {"30", Custom{Name: "test", Default: "50", Validation: "integer"}, "30"},
		"IntegerDefault": {"", Custom{Name: "test", Default: "50", Validation: "integer"}, "50"},
		"YNYes":          {"yes", Custom{Name: "test", Default: "yes", Validation: "yesorno"}, "yes"},
		"YNYesDefault":   {"", Custom{Name: "test", Default: "yes", Validation: "yesorno"}, "yes"},
		"YNY":            {"y", Custom{Name: "test", Default: "yes", Validation: "yesorno"}, "yes"},
		"YNNo":           {"no", Custom{Name: "test", Default: "yes", Validation: "yesorno"}, "no"},
		"YNn":            {"n", Custom{Name: "test", Default: "yes", Validation: "yesorno"}, "no"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			content := []byte(tc.input)

			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("error setting up environment for testing %v", err)
			}

			_, err = w.Write(content)
			if err != nil {
				t.Error(err)
			}
			w.Close()

			stdin := os.Stdin
			// Restore stdin right after the test.
			defer func() { os.Stdin = stdin }()
			os.Stdin = r

			if err := tc.custom.Collect(); err != nil {
				t.Errorf("custom.Collect failed: %v", err)
			}

			if !reflect.DeepEqual(tc.want, tc.custom.Value) {
				t.Fatalf("expected: %v, got: %v", tc.want, tc.custom.Value)
			}
		})
	}
}

func TestFindAndReadRequired(t *testing.T) {
	testdata := "test_files/configs"

	tests := map[string]struct {
		pwd       string
		terraform string
		scripts   string
		messages  string
	}{
		"Original":  {pwd: "original", terraform: ".", scripts: "scripts", messages: "messages"},
		"Perferred": {pwd: "preferred", terraform: "terraform", scripts: ".deploystack/scripts", messages: ".deploystack/messages"},
		"Configed":  {pwd: "configed", terraform: "tf", scripts: "ds/scripts", messages: "ds/messages"},
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("error setting up environment for testing %v", err)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if err := os.Chdir(fmt.Sprintf("%s/%s", testdata, tc.pwd)); err != nil {
				t.Fatalf("failed to set the wd: %v", err)
			}

			s := NewStack()

			if err := s.FindAndReadRequired(); err != nil {
				t.Fatalf("could not read config file: %s", err)
			}

			if !reflect.DeepEqual(tc.terraform, s.Config.PathTerraform) {
				t.Fatalf("expected: %s, got: %s", tc.terraform, s.Config.PathTerraform)
			}

			if !reflect.DeepEqual(tc.scripts, s.Config.PathScripts) {
				t.Fatalf("expected: %s, got: %s", tc.scripts, s.Config.PathScripts)
			}

			if !reflect.DeepEqual(tc.messages, s.Config.PathMessages) {
				t.Fatalf("expected: %s, got: %s", tc.messages, s.Config.PathMessages)
			}
		})
		if err := os.Chdir(wd); err != nil {
			t.Errorf("failed to reset the wd: %v", err)
		}
	}
}

func TestConfig(t *testing.T) {
	testdata := "test_files/configs"
	tests := map[string]struct {
		pwd      string
		want     Config
		descPath string
	}{
		"Original": {
			pwd: "original",
			want: Config{
				Title:             "Three Tier App (TODO)",
				Duration:          9,
				DocumentationLink: "https://cloud.google.com/shell/docs/cloud-shell-tutorials/deploystack/three-tier-app",
				Project:           true,
				ProjectNumber:     true,
				Region:            true,
				BillingAccount:    false,
				RegionType:        "run",
				RegionDefault:     "us-central1",
				Zone:              true,
				HardSet:           map[string]string{"basename": "three-tier-app"},
				PathTerraform:     ".",
				PathMessages:      "messages",
				PathScripts:       "scripts",
			},
			descPath: "messages/description.txt",
		},
		"YAML": {
			pwd: "preferredyaml",
			want: Config{
				Title:             "Three Tier App (TODO)",
				Duration:          9,
				DocumentationLink: "https://cloud.google.com/shell/docs/cloud-shell-tutorials/deploystack/three-tier-app",
				Project:           true,
				ProjectNumber:     true,
				Region:            true,
				BillingAccount:    false,
				RegionType:        "run",
				RegionDefault:     "us-central1",
				Zone:              true,
				HardSet:           map[string]string{"basename": "three-tier-app"},
				PathTerraform:     "terraform",
				PathMessages:      ".deploystack/messages",
				PathScripts:       ".deploystack/scripts",
				CustomSettings: []Custom{
					{
						Name:        "nodes",
						Description: "Please enter the number of nodes",
						Default:     "roles/owner|Project Owner",
						Options: []string{
							"roles/reviewer|Project Reviewer",
							"roles/owner|Project Owner",
							"roles/vison.reader|Cloud Vision Reader",
						},
					},
				},
			},
			descPath: ".deploystack/messages/description.txt",
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("error setting up environment for testing %v", err)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if err := os.Chdir(fmt.Sprintf("%s/%s", testdata, tc.pwd)); err != nil {
				t.Fatalf("failed to set the wd: %v", err)
			}

			s := NewStack()

			if err := s.FindAndReadRequired(); err != nil {
				t.Fatalf("could not read config file: %s", err)
			}

			dat, err := os.ReadFile(tc.descPath)
			if err != nil {
				t.Fatalf("could not read description file: %s", err)
			}
			tc.want.Description = string(dat)

			if !reflect.DeepEqual(tc.want, s.Config) {
				t.Fatalf("expected: %+v, got: %+v", tc.want, s.Config)
			}
		})
		if err := os.Chdir(wd); err != nil {
			t.Errorf("failed to reset the wd: %v", err)
		}
	}
}

func TestComputeNames(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
		err   error
	}{
		"http": {
			"test_files/computenames_repos/deploystack-single-vm",
			"single-vm",
			nil,
		},
		"ssh": {
			"test_files/computenames_repos/deploystack-gcs-to-bq-with-least-privileges",
			"gcs-to-bq-with-least-privileges",
			nil,
		},
		"nogit": {
			"test_files/computenames_repos/folder-no-git",
			"",
			fmt.Errorf("could not open local git directory: repository does not exist"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldWD, _ := os.Getwd()
			os.Chdir(tc.input)
			defer os.Chdir(oldWD)

			s := NewStack()
			s.FindAndReadRequired()
			err := s.Config.ComputeName()

			os.Chdir(oldWD)

			if !(tc.err == nil && err == nil) {
				if errors.Is(tc.err, err) {
					t.Fatalf("error expected: %v, got: %v", tc.err, err)
				}
			}

			if !reflect.DeepEqual(tc.want, s.Config.Name) {
				t.Fatalf("expected: %v, got: %v", tc.want, s.Config.Name)
			}
		})
	}
}

func TestPrintSetting(t *testing.T) {
	tests := map[string]struct {
		name    string
		value   string
		longest int
		want    string
	}{
		"Region": {
			"region",
			"test-a",
			6,
			"Region: [1;36mtest-a[0m\n",
		},
		"Zone": {
			"zone",
			"test-a",
			6,
			"Zone:   [1;36mtest-a[0m\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := captureOutput(func() {
				printSetting(tc.name, tc.value, tc.longest)
			})

			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected:\n|%s|\ngot:\n|%s|", tc.want, got)
			}
		})
	}
}

func TestProjectsCollect(t *testing.T) {
	testProjectsList := []ProjectWithBilling{
		{Name: "test01", ID: "test01", BillingEnabled: true},
		{Name: "test02", ID: "test02", BillingEnabled: true},
		{Name: "test03", ID: "test03", BillingEnabled: true},
		{Name: "test04", ID: "test04", BillingEnabled: true},
		{Name: "test05", ID: "test05", BillingEnabled: true},
		{Name: "test06", ID: "test06", BillingEnabled: true},
		{Name: "test07", ID: "test07", BillingEnabled: true},
		{Name: "test08", ID: "test08", BillingEnabled: true},
		{Name: "test09", ID: "test09", BillingEnabled: true},
		{Name: "test10", ID: "test10", BillingEnabled: true},
		{Name: "test11", ID: "test11", BillingEnabled: true},
		{Name: "test12", ID: "test12", BillingEnabled: true},
	}
	defaultProject := "test04"

	tests := map[string]struct {
		beforeProjects Projects
		afterProjects  Projects
		input          string
		want           string
	}{
		"simple": {
			beforeProjects: Projects{
				Items: []Project{
					{
						Name:       "test_project_1",
						UserPrompt: "Pick a first project",
					},
					{
						Name:       "test_project_2",
						UserPrompt: "Pick a second project",
					},
				},
			},
			input: "12\n\n",
			afterProjects: Projects{
				Items: []Project{
					{
						Name:       "test_project_1",
						UserPrompt: "Pick a first project",
						value:      "test11",
					},
					{
						Name:       "test_project_2",
						UserPrompt: "Pick a second project",
						value:      "test04",
					},
				},
			},
			want: `
[1;36mPick a first project[0m

[46mNOTE:[0;36m This app will make changes to the project. [0m
While those changes are reverseable, it would be better to put it in a fresh new project. 
 1) CREATE NEW PROJECT  8) test07             
 2) test01              9) test08             
 3) test02             10) test09             
 4) test03             11) test10             
[1;36m 5) test04             [0m12) test11             
 6) test05             13) test12             
 7) test06             
Choose number from list, or just [enter] for [1;36mtest04[0m
> 
[1;36mPick a second project[0m

[46mNOTE:[0;36m This app will make changes to the project. [0m
While those changes are reverseable, it would be better to put it in a fresh new project. 
 1) CREATE NEW PROJECT  7) test06             
 2) test01              8) test07             
 3) test02              9) test08             
 4) test03             10) test09             
[1;36m 5) test04             [0m11) test10             
 6) test05             12) test12             
Choose number from list, or just [enter] for [1;36mtest04[0m
> `,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			origStdin := os.Stdin

			testStdin, err := ioutil.TempFile("", "")
			if err != nil {
				t.Fatal(err)
			}

			defer testStdin.Close()

			os.Stdin = testStdin

			if _, err := io.WriteString(testStdin, tc.input); err != nil {
				t.Fatalf("expected: no error got: error(%s)", err)
			}

			if _, err = testStdin.Seek(0, os.SEEK_SET); err != nil {
				t.Fatalf("expected: no error got: error(%s)", err)
			}

			got := captureOutput(func() {
				tc.beforeProjects.Collect(testProjectsList, defaultProject)
			})

			if !reflect.DeepEqual(tc.afterProjects, tc.beforeProjects) {
				t.Fatalf("expected:\n%v\ngot:\n%v", tc.afterProjects, tc.beforeProjects)
			}

			if !reflect.DeepEqual(tc.want, string(got)) {
				os.Stdin = origStdin
				fmt.Println(diff.Diff(got, tc.want))
				t.Fatal("Should be the same")
			}
			os.Stdin = origStdin
		})
	}
}

func TestDomainRegistrarContactReadYAML(t *testing.T) {
	tests := map[string]struct {
		file string
		want ContactData
		err  error
	}{
		"simple": {
			file: "test_files/contact_sample.yaml",
			want: ContactData{DomainRegistrarContact{
				"you@example.com",
				"+1 555 555 1234",
				PostalAddress{
					"US",
					"94105",
					"CA",
					"San Francisco",
					[]string{"345 Spear Street"},
					[]string{"Your Name"},
				},
			}},
			err: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewContactDataFromFile(tc.file)

			if err != tc.err {
				if err != nil && tc.err != nil && err.Error() != tc.err.Error() {
					t.Fatalf("expected: error(%s) got: error(%s)", tc.err, err)
				}
			}

			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %v got: %v", tc.want, got)
			}
		})
	}
}

func TestDomainContact(t *testing.T) {
	contact := &domainspb.ContactSettings_Contact{
		PostalAddress: &postaladdress.PostalAddress{
			RegionCode:         "US",
			PostalCode:         "94105",
			AdministrativeArea: "CA",
			Locality:           "San Francisco",
			AddressLines:       []string{"345 Spear Street"},
			Recipients:         []string{"Your Name"},
		},
		Email:       "you@example.com",
		PhoneNumber: "+1 555 555 1234",
	}

	tests := map[string]struct {
		input ContactData
		want  domainspb.ContactSettings
		err   error
	}{
		"simple": {
			input: ContactData{DomainRegistrarContact{
				"you@example.com",
				"+1 555 555 1234",
				PostalAddress{
					"US",
					"94105",
					"CA",
					"San Francisco",
					[]string{"345 Spear Street"},
					[]string{"Your Name"},
				},
			}},
			want: domainspb.ContactSettings{
				Privacy:           domainspb.ContactPrivacy_PRIVATE_CONTACT_DATA,
				RegistrantContact: contact,
				AdminContact:      contact,
				TechnicalContact:  contact,
			},
			err: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tc.input.DomainContact()

			if err != tc.err {
				if err != nil && tc.err != nil && err.Error() != tc.err.Error() {
					t.Fatalf("expected: error(%s) got: error(%s)", tc.err, err)
				}
			}

			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %+v got: %+v", tc.want, got)
			}
		})
	}
}
