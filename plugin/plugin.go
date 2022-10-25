// Copyright 2019 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/converter"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var documentSeperator = regexp.MustCompile(`(?m)^---($| )`)

const secretTemplate = `%s
---
kind: secret
name: %q
get:
  name: %q
  path: ""`

// New returns a new conversion plugin.
func New() converter.Plugin {
	return &plugin{}
}

type plugin struct{}

func (p *plugin) Convert(ctx context.Context, req *converter.Request) (*drone.Config, error) {
	// get the configuration file from the request.
	configs := splitConfigs(req.Config.Data)

	// load secrets
	secrets := map[string]struct{}{}
	for _, config := range configs {
		err := findSecrets(config, &secrets)
		if err != nil {
			return nil, fmt.Errorf("unable to parse configuration as yaml: %w", err)
		}
	}
	logrus.Debugf("%s injected: %+v", req.Repo.Slug, secrets)

	// template and append to data
	secretYaml := ""
	for secret := range secrets {
		secretYaml = fmt.Sprintf(secretTemplate, secretYaml, secret, secret)
	}
	data := strings.ReplaceAll(req.Config.Data, "\n...\n", "") + secretYaml

	// returns the modified configuration file.
	return &drone.Config{
		Data: data,
	}, nil
}

// splitConfigs splits multiple drone pipelines into an array of singles
func splitConfigs(config string) []string {
	configs := documentSeperator.Split(config, -1)
	for i := range configs {
		configs[i] = strings.TrimSpace(configs[i])
	}
	return configs
}

// findSecrets gathers all secret names in a drone pipeline
func findSecrets(config string, output *map[string]struct{}) error {
	var data yaml.Node
	err := yaml.Unmarshal([]byte(config), &data)
	if err != nil {
		return err
	}
	findSecretsInNode(&data, output)
	return nil
}

// findSecretsInNode parses a yaml tree for a `from_secret: string_value`
func findSecretsInNode(node *yaml.Node, output *map[string]struct{}) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, content := range node.Content {
			findSecretsInNode(content, output)
		}
	case yaml.SequenceNode:
		for _, content := range node.Content {
			findSecretsInNode(content, output)
		}
	case yaml.MappingNode:
		// an array of key, value, key, value, ...
		//             ----------, ----------, ...
		for i := 0; i < (len(node.Content) - 1); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if key.Value == "from_secret" && value.Kind == yaml.ScalarNode {
				(*output)[value.Value] = struct{}{}
			} else {
				findSecretsInNode(value, output)
			}
		}
	case yaml.AliasNode:
	case yaml.ScalarNode:
	}
}
