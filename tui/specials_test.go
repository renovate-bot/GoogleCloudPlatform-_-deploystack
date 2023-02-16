package tui

import (
	"testing"

	"github.com/GoogleCloudPlatform/deploystack"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewProjectCreator(t *testing.T) {
	tests := map[string]struct {
		key        string
		outputFile string
	}{
		"basic": {
			key:        "project_id",
			outputFile: "testdata/project_creator_basic.txt",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q := getTestQueue(appTitle, "test")
			out := newProjectCreator(tc.key)
			q.add(out)

			got := out.View()
			want := readTestFile(tc.outputFile)

			if want != got {
				writeDebugFile(got, tc.outputFile)
				t.Fatalf("text wasn't the same")
			}
		})
	}
}

func TestNewProjectSelector(t *testing.T) {
	tests := map[string]struct {
		key          string
		listLabel    string
		preProcessor tea.Cmd
		outputFile   string
		update       bool
	}{
		"waiting": {
			key:        "project_id",
			listLabel:  "Selecte a project to use",
			outputFile: "testdata/project_selector_waiting.txt",
		},
		"updated": {
			key:        "project_id",
			listLabel:  "Selecte a project to use",
			outputFile: "testdata/project_selector_updated.txt",
			update:     true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q := getTestQueue(appTitle, "test")

			out := newProjectSelector(tc.key, tc.listLabel, getProjects(&q))
			q.add(&out)

			if tc.update {
				cmd := out.Init()
				for i := 0; i < 2; i++ {

					msg := cmd()

					switch v := msg.(type) {
					case tea.BatchMsg:
						msgs := msg.(tea.BatchMsg)

						for _, v2 := range msgs {
							var tmp tea.Model
							tmp, cmd = out.Update(v2())
							out = tmp.(picker)
						}
					default:
						var tmp tea.Model
						tmp, cmd = out.Update(v)
						out = tmp.(picker)
					}

				}

			}

			got := out.View()
			want := readTestFile(tc.outputFile)

			if want != got {
				writeDebugFile(got, tc.outputFile)
				t.Fatalf("text wasn't the same")
			}
		})
	}
}

func TestNewCustom(t *testing.T) {
	tests := map[string]struct {
		c          deploystack.Custom
		outputFile string
	}{
		"basic": {
			c: deploystack.Custom{
				Name:        "test",
				Description: "A test option",
				Default:     "Test",
			},
			outputFile: "testdata/custom_basic.txt",
		},
		"phone": {
			c: deploystack.Custom{
				Name:        "test",
				Description: "A test phone",
				Default:     "1-555-555-4040",
				Validation:  validationPhoneNumber,
			},
			outputFile: "testdata/custom_phone.txt",
		},
		"yesorno": {
			c: deploystack.Custom{
				Name:        "test",
				Description: "Yay or Nay",
				Default:     "Yes",
				Validation:  validationYesOrNo,
			},
			outputFile: "testdata/custom_yesorno.txt",
		},
		"integer": {
			c: deploystack.Custom{
				Name:        "test",
				Description: "a number",
				Default:     "5",
				Validation:  validationInteger,
			},
			outputFile: "testdata/custom_integer.txt",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q := getTestQueue(appTitle, "test")
			out := newCustom(tc.c)
			q.add(out)

			got := out.View()
			want := readTestFile(tc.outputFile)

			if want != got {
				writeDebugFile(got, tc.outputFile)
				t.Fatalf("text wasn't the same")
			}
		})
	}
}

func TestQueueBatch(t *testing.T) {
	tests := map[string]struct {
		f     func(*Queue)
		count int
		keys  []string
	}{
		"region": {
			f:     newRegion,
			count: 1,
			keys:  []string{"region"},
		},
		"zone": {
			f:     newZone,
			count: 1,
			keys:  []string{"zone"},
		},

		"domain": {
			f:     newDomain,
			count: 10,
			keys: []string{
				"domain",
				"domain_email",
				"domain_phone",
				"domain_country",
				"domain_postalcode",
				"domain_state",
				"domain_city",
				"domain_address",
				"domain_name",
				"domain_consent",
			},
		},

		"GCEInstance": {
			f:     newGCEInstance,
			count: 12,
			keys: []string{
				"gce-use-defaults",
				"instance-name",
				"region",
				"zone",
				"instance-machine-type-family",
				"instance-machine-type",
				"instance-image-project",
				"instance-image-family",
				"instance-image",
				"instance-disktype",
				"instance-disksize",
				"instance-webserver",
			},
		},
		"MachineTypeManager": {
			f:     newMachineTypeManager,
			count: 2,
			keys: []string{
				"instance-machine-type-family",
				"instance-machine-type",
			},
		},

		"DiskImageManager": {
			f:     newDiskImageManager,
			count: 3,
			keys: []string{
				"instance-image-project",
				"instance-image-family",
				"instance-image",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q := getTestQueue(appTitle, "test")
			tc.f(&q)

			if tc.count != len(q.models) {
				t.Fatalf("count - want '%d' got '%d'", tc.count, len(q.models))
			}

			for _, v := range tc.keys {
				q.removeModel(v)
			}

			if 0 != len(q.models) {
				t.Logf("Models remain")
				for _, v := range q.models {
					t.Logf("%s", v.getKey())
				}

				t.Fatalf("key check - want '%d' got '%d'", 0, len(q.models))

			}
		})
	}
}

func TestCustomPages(t *testing.T) {
	tests := map[string]struct {
		config string
		count  int
		keys   []string
	}{
		"region": {
			config: "testdata/config_multicustom.yaml",
			count:  5,
			keys: []string{
				"nodes",
				"label",
				"location",
				"budgetamount",
				"yesorno",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q := getTestQueue(appTitle, "test")

			s := readTestFile(tc.config)

			config, err := deploystack.NewConfigYAML([]byte(s))
			if err != nil {
				t.Fatalf("could not read in config %s:", err)
			}
			q.stack.Config = config

			newCustomPages(&q)

			if tc.count != len(q.models) {
				t.Logf("Models ")
				for _, v := range q.models {
					t.Logf("%s", v.getKey())
				}
				t.Fatalf("count - want '%d' got '%d'", tc.count, len(q.models))
			}

			for _, v := range tc.keys {
				q.removeModel(v)
			}

			if 0 != len(q.models) {
				t.Logf("Models remain")
				for _, v := range q.models {
					t.Logf("%s", v.getKey())
				}

				t.Fatalf("key check - want '%d' got '%d'", 0, len(q.models))

			}
		})
	}
}