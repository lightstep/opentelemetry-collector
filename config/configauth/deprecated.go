// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configauth // import "go.opentelemetry.io/collector/config/configauth"

import (
	"go.opentelemetry.io/collector/extension/auth"
	"go.opentelemetry.io/collector/extension/auth/authtest"
)

// Deprecated: [v0.67.0] Use auth.Client
type ClientAuthenticator = auth.Client

// Deprecated: [v0.67.0] Use auth.ClientOption
type ClientOption = auth.ClientOption

// Deprecated: [v0.67.0] Use auth.WithClientStart
var WithClientStart = auth.WithClientStart

// Deprecated: [v0.67.0] Use auth.WithClientShutdown
var WithClientShutdown = auth.WithClientShutdown

// Deprecated: [v0.67.0] Use auth.WithClientRoundTripper
var WithClientRoundTripper = auth.WithClientRoundTripper

// Deprecated: [v0.67.0] Use auth.WithPerRPCCredentials
var WithPerRPCCredentials = auth.WithPerRPCCredentials

// Deprecated: [v0.67.0] Use auth.NewClient
var NewClientAuthenticator = auth.NewClient

// Deprecated: [v0.67.0] Use auth.Server
type ServerAuthenticator = auth.Server

// Deprecated: [v0.67.0] Use auth.AuthenticateFunc
type AuthenticateFunc = auth.AuthenticateFunc

// Deprecated: [v0.67.0] Use auth.Option
type Option = auth.Option

// Deprecated: [v0.67.0] Use auth.WithAuthenticate
var WithAuthenticate = auth.WithAuthenticate

// Deprecated: [v0.67.0] Use auth.WithStart
var WithStart = auth.WithStart

// Deprecated: [v0.67.0] Use auth.WithShutdown
var WithShutdown = auth.WithShutdown

// Deprecated: [v0.67.0] Use auth.NewServer
var NewServerAuthenticator = auth.NewServer

// Deprecated: [v0.67.0] Use auth.MockClient
type MockClientAuthenticator = authtest.MockClient
