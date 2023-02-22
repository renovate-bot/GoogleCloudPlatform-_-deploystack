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
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/deploystack/gcloud"
)

func TestCacheContact(t *testing.T) {
	tests := map[string]struct {
		in  gcloud.ContactData
		err error
	}{
		"basic": {
			in: gcloud.ContactData{
				AllContacts: gcloud.DomainRegistrarContact{
					Email: "test@example.com",
					Phone: "+155555551212",
					PostalAddress: gcloud.PostalAddress{
						RegionCode:         "US",
						PostalCode:         "94502",
						AdministrativeArea: "CA",
						Locality:           "San Francisco",
						AddressLines:       []string{"345 Spear Street"},
						Recipients:         []string{"Googler"},
					},
				},
			},
			err: nil,
		},
		"err": {
			in:  gcloud.ContactData{},
			err: fmt.Errorf("stat contact.yaml: no such file or directory"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			CacheContact(tc.in)

			if tc.err == nil {
				if _, err := os.Stat(contactfile); errors.Is(err, os.ErrNotExist) {
					t.Fatalf("expected no error,  got: %+v", err)
				}
			} else {
				if _, err := os.Stat(contactfile); err.Error() != tc.err.Error() {
					t.Fatalf("expected %+v, got: %+v", tc.err, err)
				}

			}

			os.Remove(contactfile)

		})
	}
}

func TestNewContactDataFromFile(t *testing.T) {
	tests := map[string]struct {
		in   string
		want gcloud.ContactData
		err  error
	}{
		"basic": {
			in: "test_files/contact/contact.yaml",
			want: gcloud.ContactData{
				AllContacts: gcloud.DomainRegistrarContact{
					Email: "test@example.com",
					Phone: "+155555551212",
					PostalAddress: gcloud.PostalAddress{
						RegionCode:         "US",
						PostalCode:         "94502",
						AdministrativeArea: "CA",
						Locality:           "San Francisco",
						AddressLines:       []string{"345 Spear Street"},
						Recipients:         []string{"Googler"},
					},
				},
			},
			err: nil,
		},
		"error": {
			in:   "test_files/contact/noexists.yaml",
			want: gcloud.ContactData{},
			err:  fmt.Errorf("open test_files/contact/noexists.yaml: no such file or directory"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewContactDataFromFile(tc.in)

			if tc.err == nil {

				if err != nil {
					t.Fatalf("expected no error,  got: %+v", err)
				}

				if !reflect.DeepEqual(tc.want, got) {
					t.Fatalf("expected: %+v, got: %+v", tc.want, got)
				}

			} else {
				if err.Error() != tc.err.Error() {
					t.Fatalf("expected %+v, got: %+v", tc.err, err)
				}
			}

		})
	}
}

func TestCheckForContact(t *testing.T) {
	tests := map[string]struct {
		in   string
		want gcloud.ContactData
	}{
		"basic": {
			in: "test_files/contact/contact.yaml",
			want: gcloud.ContactData{
				AllContacts: gcloud.DomainRegistrarContact{
					Email: "test@example.com",
					Phone: "+155555551212",
					PostalAddress: gcloud.PostalAddress{
						RegionCode:         "US",
						PostalCode:         "94502",
						AdministrativeArea: "CA",
						Locality:           "San Francisco",
						AddressLines:       []string{"345 Spear Street"},
						Recipients:         []string{"Googler"},
					},
				},
			},
		},

		"empty": {
			in:   contactfile,
			want: gcloud.ContactData{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			oldContactFile := contactfile
			contactfile = tc.in

			got := CheckForContact()
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %+v, got: %+v", tc.want, got)
			}

			contactfile = oldContactFile
		})
	}
}
