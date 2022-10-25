// Copyright 2019 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/converter"
)

func TestSplitConfigs(t *testing.T) {
	y := `
a
---
b
`
	result := splitConfigs(y)
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Fatalf("splitting documents produced unexpected output: %+v", result)
	}
}

func TestFindSecrets(t *testing.T) {
	y := `---
kind: pipeline
name: default

steps:
- name: build
  image: alpine
  environment:
    USERNAME:
      from_secret: secret_username
    PASSWORD:
      from_secret: secret_password
`
	secrets := map[string]struct{}{}
	err := findSecrets(y, &secrets)
	_ = err
	_, hasUsername := secrets["secret_username"]
	_, hasPassword := secrets["secret_password"]

	if len(secrets) != 2 || !hasUsername || !hasPassword {
		t.Fatalf("unable to find all secrets: %+v", secrets)

	}

}

func TestConvert(t *testing.T) {
	y := `---
kind: pipeline
name: default

steps:
- name: build
  image: alpine
  environment:
    USERNAME:
      from_secret: secret_username
    PASSWORD:
      from_secret: secret_password
`
	ctx := context.Background()
	req := converter.Request{
		Config: drone.Config{
			Data: y,
		},
	}

	plugin := New()
	config, err := plugin.Convert(ctx, &req)
	if err != nil {
		t.Fatalf("error parsing config: %s", err)
	}

	should := `---
kind: pipeline
name: default

steps:
- name: build
  image: alpine
  environment:
    USERNAME:
      from_secret: secret_username
    PASSWORD:
      from_secret: secret_password

---
kind: secret
name: "secret_username"
get:
  name: "secret_username"
  path: ""
---
kind: secret
name: "secret_password"
get:
  name: "secret_password"
  path: ""`
	if config.Data != should {
		t.Fatalf("invalid pipeline result: \n%s\n", config.Data)
	}

}
