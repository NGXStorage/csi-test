/*
Copyright 2023 The Kubernetes Authors.

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

package utils

import (
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// Protobuf returns a Matcher that relies upon proto.Equal to compare Protobuf messages
// Example usage with mocked request:
//
// example.EXPECT().ExampleRequest(Protobuf(requestMsg)).Return(responseMsg, nil).AnyTimes()
func Protobuf(msg proto.Message) gomock.Matcher {
	return &ProtobufMatcher{msg}
}

type ProtobufMatcher struct {
	msg proto.Message
}

var _ gomock.Matcher = &ProtobufMatcher{}

func (p *ProtobufMatcher) Matches(x interface{}) bool {
	otherMsg, ok := x.(proto.Message)
	if !ok {
		return false
	}
	return proto.Equal(p.msg, otherMsg)
}

func (p *ProtobufMatcher) String() string {
	return prototext.Format(p.msg)
}
