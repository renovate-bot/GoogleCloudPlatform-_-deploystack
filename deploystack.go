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

// Package deploystack provides a series of interfaces for getting Google Cloud
// settings and configurations for use with DeplyStack
package deploystack

import (
	"fmt"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/deploystack/gcloud"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

var (
	opts             = option.WithCredentialsFile("")
	credspath        = ""
	defaultUserAgent = "deploystack"
	contactfile      = "contact.yaml"
)

// Init initializes a Deploystack stack by looking on teh local file system
func Init() (*Stack, error) {
	s := NewStack()

	if err := s.FindAndReadRequired(); err != nil {
		return &s, fmt.Errorf("could not read config file: %s", err)
	}

	if s.Config.Name == "" {
		if err := s.Config.ComputeName(); err != nil {
			return &s, fmt.Errorf("could retrieve name of stack: %s", err)
		}
		s.AddSetting("stack_name", s.Config.Name)
	}

	return &s, nil
}

// func handleProcessError(err error) {
// 	fmt.Printf("\n\n%sThere was an issue collecting the information it takes to run this application.                             %s\n\n", TERMREDREV, TERMCLEAR)
// 	fmt.Printf("%sYou can try again by typing %sdeploystack install%s at the command prompt  %s\n\n", TERMREDB, TERMREDREV, TERMCLEAR+TERMREDB, TERMCLEAR)
// 	fmt.Printf("%sIf the issue persists, please report at https://github.com/GoogleCloudPlatform/deploystack/issues %s\n\n", TERMREDB, TERMCLEAR)

// 	fmt.Printf("Extra diagnostic information:\n")

// 	if strings.Contains(err.Error(), "invalid token JSON from metadata") {
// 		fmt.Printf("timed out waiting for API activation, you must authorize API use to continue \n")
// 	}

// 	fmt.Println(err)
// 	os.Exit(1)
// }

// NewContactDataFromFile generates a new ContactData from a cached yaml file
func NewContactDataFromFile(file string) (gcloud.ContactData, error) {
	c := gcloud.NewContactData()

	dat, err := os.ReadFile(file)
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(dat, &c)
	if err != nil {
		return c, err
	}

	return c, nil
}

// CheckForContact checks the local file system for a file containg domain
// registar contact info
func CheckForContact() gcloud.ContactData {
	contact := gcloud.ContactData{}
	if _, err := os.Stat(contactfile); err == nil {
		contact, err = NewContactDataFromFile(contactfile)
		if err != nil {
			log.Printf("domain registrar contact not cached")
		}
	}
	return contact
}

// CacheContact writes a file containg domain registar contact info to disk
// if it exists
func CacheContact(i interface{}) {
	switch v := i.(type) {
	case gcloud.ContactData:
		if v.AllContacts.Email != "" {
			yaml, err := v.YAML()
			if err != nil {
				log.Printf("could not convert contact to yaml: %s", err)
			}

			if err := os.WriteFile(contactfile, []byte(yaml), 0o644); err != nil {
				log.Printf("could not write contact to file: %s", err)
			}
		}
	}
}
