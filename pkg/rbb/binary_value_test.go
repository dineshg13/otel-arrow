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

package rbb

import (
	"github.com/apache/arrow/go/arrow"
	"testing"
)

func TestCoerceFromString(t *testing.T) {
	// Test coerce on a scalar value
	dataType1 := (&String{Value: "true"}).DataType()
	dataType2 := (&I8{Value: 1}).DataType()
	dataType := CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&U8{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&I16{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&U16{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&I32{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&U32{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&I64{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&U64{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&Bool{Value: true}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}

	dataType1 = (&String{Value: "true"}).DataType()
	dataType2 = (&String{Value: "bla"}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.STRING {
		t.Errorf("Expected STRING, got %v", dataType.ID())
	}
}

func TestCoerceFromBinary(t *testing.T) {
	// Test coerce on a scalar value
	dataType1 := (&Binary{Value: []byte("true")}).DataType()
	dataType2 := (&I8{Value: 1}).DataType()
	dataType := CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&U8{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&I16{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&U16{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&I32{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&U32{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&I64{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&U64{Value: 1}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&Bool{Value: true}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}

	dataType1 = (&Binary{Value: []byte("true")}).DataType()
	dataType2 = (&String{Value: "bla"}).DataType()
	dataType = CoerceDataTypes(dataType1, dataType2)
	if dataType.ID() != arrow.BINARY {
		t.Errorf("Expected BINARY, got %v", dataType.ID())
	}
}