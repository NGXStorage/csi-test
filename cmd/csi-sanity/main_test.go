/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"flag"
	"io"
	stdtesting "testing"

	"github.com/kubernetes-csi/csi-test/v5/pkg/sanity"
)

func TestSharedArrayFlag(t *stdtesting.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    bool
		invalid bool
	}{
		{name: "dedicated by default"},
		{name: "shared", args: []string{"--shared-array"}, want: true},
		{name: "explicit dedicated", args: []string{"--shared-array=false"}},
		{name: "invalid", args: []string{"--shared-array=invalid"}, invalid: true},
	} {
		t.Run(tc.name, func(t *stdtesting.T) {
			previous := flag.CommandLine
			flag.CommandLine = flag.NewFlagSet("csi-sanity", flag.ContinueOnError)
			t.Cleanup(func() { flag.CommandLine = previous })
			flag.CommandLine.SetOutput(io.Discard)
			flag.CommandLine.Var(previous.Lookup("ginkgo.junit-report").Value, "ginkgo.junit-report", "")
			config := sanity.NewTestConfig()
			registerCSIFlags(&config)
			err := flag.CommandLine.Parse(tc.args)
			if (err != nil) != tc.invalid || config.SharedArray != tc.want {
				t.Fatalf("args %v: SharedArray=%v, parse error=%v", tc.args, config.SharedArray, err)
			}
		})
	}
}
