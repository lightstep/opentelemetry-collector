// Copyright The OpenTelemetry Authors
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

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package ptrace

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// ResourceSpans is a collection of spans from a Resource.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewResourceSpans function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceSpans struct {
	orig *otlptrace.ResourceSpans
}

func newResourceSpans(orig *otlptrace.ResourceSpans) ResourceSpans {
	return ResourceSpans{orig}
}

// NewResourceSpans creates a new empty ResourceSpans.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewResourceSpans() ResourceSpans {
	return newResourceSpans(&otlptrace.ResourceSpans{})
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms ResourceSpans) MoveTo(dest ResourceSpans) {
	*dest.orig = *ms.orig
	*ms.orig = otlptrace.ResourceSpans{}
}

// Resource returns the resource associated with this ResourceSpans.
func (ms ResourceSpans) Resource() pcommon.Resource {
	return pcommon.Resource(internal.NewResource(&ms.orig.Resource))
}

// SchemaUrl returns the schemaurl associated with this ResourceSpans.
func (ms ResourceSpans) SchemaUrl() string {
	return ms.orig.SchemaUrl
}

// SetSchemaUrl replaces the schemaurl associated with this ResourceSpans.
func (ms ResourceSpans) SetSchemaUrl(v string) {
	ms.orig.SchemaUrl = v
}

// ScopeSpans returns the ScopeSpans associated with this ResourceSpans.
func (ms ResourceSpans) ScopeSpans() ScopeSpansSlice {
	return newScopeSpansSlice(&ms.orig.ScopeSpans)
}

// CopyTo copies all properties from the current struct overriding the destination.
func (ms ResourceSpans) CopyTo(dest ResourceSpans) {
	ms.Resource().CopyTo(dest.Resource())
	dest.SetSchemaUrl(ms.SchemaUrl())
	ms.ScopeSpans().CopyTo(dest.ScopeSpans())
}
