/*
 * Copyright The OpenTelemetry Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package arrow

// Attributes record builder for 16-bit Parent IDs.

import (
	"errors"
	"sort"

	"github.com/apache/arrow/go/v12/arrow"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/f5/otel-arrow-adapter/pkg/otel/common"
	"github.com/f5/otel-arrow-adapter/pkg/otel/common/schema"
	"github.com/f5/otel-arrow-adapter/pkg/otel/common/schema/builder"
	"github.com/f5/otel-arrow-adapter/pkg/otel/constants"
	"github.com/f5/otel-arrow-adapter/pkg/werror"
)

var (
	// AttrsSchema16 is the Arrow schema used to represent attribute records
	// with 16-bit Parent IDs.
	// This schema doesn't use the Arrow union type to make Parquet conversion
	// more direct.
	AttrsSchema16 = arrow.NewSchema([]arrow.Field{
		{Name: constants.ParentID, Type: arrow.PrimitiveTypes.Uint16},
		{Name: constants.AttributeKey, Type: arrow.BinaryTypes.String, Metadata: schema.Metadata(schema.Dictionary8)},
		{Name: constants.AttributeType, Type: arrow.PrimitiveTypes.Uint8},
		{Name: constants.AttributeStr, Type: arrow.BinaryTypes.String, Metadata: schema.Metadata(schema.Dictionary16)},
		{Name: constants.AttributeInt, Type: arrow.PrimitiveTypes.Int64, Metadata: schema.Metadata(schema.Optional, schema.Dictionary16)},
		{Name: constants.AttributeDouble, Type: arrow.PrimitiveTypes.Float64, Metadata: schema.Metadata(schema.Optional)},
		{Name: constants.AttributeBool, Type: arrow.FixedWidthTypes.Boolean, Metadata: schema.Metadata(schema.Optional)},
		{Name: constants.AttributeBytes, Type: arrow.BinaryTypes.Binary, Metadata: schema.Metadata(schema.Optional, schema.Dictionary16)},
		{Name: constants.AttributeSer, Type: arrow.BinaryTypes.Binary, Metadata: schema.Metadata(schema.Optional, schema.Dictionary16)},
	}, nil)
)

type (
	Attrs16Builder struct {
		released bool

		builder *builder.RecordBuilderExt // Record builder

		pib   *builder.Uint16Builder
		keyb  *builder.StringBuilder
		typeb *builder.Uint8Builder
		strb  *builder.StringBuilder
		i64b  *builder.Int64Builder
		f64b  *builder.Float64Builder
		boolb *builder.BooleanBuilder
		binb  *builder.BinaryBuilder
		serb  *builder.BinaryBuilder

		accumulator *Attributes16Accumulator
		payloadType *PayloadType
	}

	Attrs16ByNothing          struct{}
	Attrs16ByParentIdKeyValue struct {
		prevParentID uint16
	}
	Attrs16ByKeyParentIdValue struct {
		prevParentID uint16
		prevKey      string
		prevValue    *pcommon.Value
	}
	Attrs16ByKeyValueParentId struct {
		prevParentID uint16
		prevKey      string
		prevValue    *pcommon.Value
	}
)

func NewAttrs16Builder(rBuilder *builder.RecordBuilderExt, payloadType *PayloadType, sorter Attrs16Sorter) *Attrs16Builder {
	b := &Attrs16Builder{
		released:    false,
		builder:     rBuilder,
		accumulator: NewAttributes16Accumulator(sorter),
		payloadType: payloadType,
	}
	b.init()
	return b
}

func NewAttrs16BuilderWithEncoding(rBuilder *builder.RecordBuilderExt, payloadType *PayloadType, config *Attrs16Config) *Attrs16Builder {
	b := &Attrs16Builder{
		released:    false,
		builder:     rBuilder,
		accumulator: NewAttributes16Accumulator(config.Sorter),
		payloadType: payloadType,
	}

	b.init()
	return b
}

func (b *Attrs16Builder) init() {
	b.pib = b.builder.Uint16Builder(constants.ParentID)
	b.keyb = b.builder.StringBuilder(constants.AttributeKey)
	b.typeb = b.builder.Uint8Builder(constants.AttributeType)
	b.strb = b.builder.StringBuilder(constants.AttributeStr)
	b.i64b = b.builder.Int64Builder(constants.AttributeInt)
	b.f64b = b.builder.Float64Builder(constants.AttributeDouble)
	b.boolb = b.builder.BooleanBuilder(constants.AttributeBool)
	b.binb = b.builder.BinaryBuilder(constants.AttributeBytes)
	b.serb = b.builder.BinaryBuilder(constants.AttributeSer)
}

func (b *Attrs16Builder) Accumulator() *Attributes16Accumulator {
	return b.accumulator
}

func (b *Attrs16Builder) TryBuild() (record arrow.Record, err error) {
	if b.released {
		return nil, werror.Wrap(ErrBuilderAlreadyReleased)
	}

	b.accumulator.Sort()

	for _, attr := range b.accumulator.attrs {
		b.pib.Append(b.accumulator.sorter.Encode(attr.ParentID, attr.Key, attr.Value))
		b.keyb.Append(attr.Key)

		switch attr.Value.Type() {
		case pcommon.ValueTypeStr:
			b.typeb.Append(uint8(pcommon.ValueTypeStr))
			b.strb.Append(attr.Value.Str())
			b.i64b.AppendNull()
			b.f64b.AppendNull()
			b.boolb.AppendNull()
			b.binb.AppendNull()
			b.serb.AppendNull()
		case pcommon.ValueTypeInt:
			b.typeb.Append(uint8(pcommon.ValueTypeInt))
			b.i64b.Append(attr.Value.Int())
			b.strb.AppendNull()
			b.f64b.AppendNull()
			b.boolb.AppendNull()
			b.binb.AppendNull()
			b.serb.AppendNull()
		case pcommon.ValueTypeDouble:
			b.typeb.Append(uint8(pcommon.ValueTypeDouble))
			b.f64b.Append(attr.Value.Double())
			b.strb.AppendNull()
			b.i64b.AppendNull()
			b.boolb.AppendNull()
			b.binb.AppendNull()
			b.serb.AppendNull()
		case pcommon.ValueTypeBool:
			b.typeb.Append(uint8(pcommon.ValueTypeBool))
			b.boolb.Append(attr.Value.Bool())
			b.strb.AppendNull()
			b.i64b.AppendNull()
			b.f64b.AppendNull()
			b.binb.AppendNull()
			b.serb.AppendNull()
		case pcommon.ValueTypeBytes:
			b.typeb.Append(uint8(pcommon.ValueTypeBytes))
			b.binb.Append(attr.Value.Bytes().AsRaw())
			b.strb.AppendNull()
			b.i64b.AppendNull()
			b.f64b.AppendNull()
			b.boolb.AppendNull()
			b.serb.AppendNull()
		case pcommon.ValueTypeSlice:
			cborData, err := common.Serialize(attr.Value)
			if err != nil {
				break
			}
			b.typeb.Append(uint8(pcommon.ValueTypeSlice))
			b.serb.Append(cborData)
			b.strb.AppendNull()
			b.i64b.AppendNull()
			b.f64b.AppendNull()
			b.boolb.AppendNull()
			b.binb.AppendNull()
		case pcommon.ValueTypeMap:
			cborData, err := common.Serialize(attr.Value)
			if err != nil {
				break
			}
			b.typeb.Append(uint8(pcommon.ValueTypeMap))
			b.serb.Append(cborData)
			b.strb.AppendNull()
			b.i64b.AppendNull()
			b.f64b.AppendNull()
			b.boolb.AppendNull()
			b.binb.AppendNull()
		}
	}

	record, err = b.builder.NewRecord()
	if err != nil {
		b.init()
	}

	return
}

func (b *Attrs16Builder) IsEmpty() bool {
	return b.accumulator.IsEmpty()
}

func (b *Attrs16Builder) Build() (arrow.Record, error) {
	schemaNotUpToDateCount := 0

	var record arrow.Record
	var err error

	// Loop until the record is built successfully.
	// Intermediaries steps may be required to update the schema.
	for {
		record, err = b.TryBuild()
		if err != nil {
			if record != nil {
				record.Release()
			}

			switch {
			case errors.Is(err, schema.ErrSchemaNotUpToDate):
				schemaNotUpToDateCount++
				if schemaNotUpToDateCount > 5 {
					panic("Too many consecutive schema updates. This shouldn't happen.")
				}
			default:
				return nil, werror.Wrap(err)
			}
		} else {
			break
		}
	}

	return record, werror.Wrap(err)
}

func (b *Attrs16Builder) SchemaID() string {
	return b.builder.SchemaID()
}

func (b *Attrs16Builder) Schema() *arrow.Schema {
	return b.builder.Schema()
}

func (b *Attrs16Builder) PayloadType() *PayloadType {
	return b.payloadType
}

func (b *Attrs16Builder) Reset() {
	b.accumulator.Reset()
}

// Release releases the memory allocated by the builder.
func (b *Attrs16Builder) Release() {
	if !b.released {
		b.builder.Release()
		b.released = true
	}
}

func (b *Attrs16Builder) ShowSchema() {
	b.builder.ShowSchema()
}

// Sorts the attributes by parentID, key, and value
// ================================================

func SortByParentIdKeyValueAttr16() *Attrs16ByParentIdKeyValue {
	return &Attrs16ByParentIdKeyValue{}
}

func (s *Attrs16ByParentIdKeyValue) Sort(attrs []Attr16) {
	sort.Slice(attrs, func(i, j int) bool {
		attrsI := attrs[i]
		attrsJ := attrs[j]
		if attrsI.ParentID == attrsJ.ParentID {
			if attrsI.Key == attrsJ.Key {
				return IsLess(attrsI.Value, attrsJ.Value)
			} else {
				return attrsI.Key < attrsJ.Key
			}
		} else {
			return attrsI.ParentID < attrsJ.ParentID
		}
	})
}

func (s *Attrs16ByParentIdKeyValue) Encode(parentID uint16, _ string, _ pcommon.Value) uint16 {
	delta := parentID - s.prevParentID
	s.prevParentID = parentID
	return delta
}

func (s *Attrs16ByParentIdKeyValue) Reset() {
	s.prevParentID = 0
}

// No sorting
// ==========

func UnsortedAttrs16() *Attrs16ByNothing {
	return &Attrs16ByNothing{}
}

func (s *Attrs16ByNothing) Sort(_ []Attr16) {
	// Do nothing
}

func (s *Attrs16ByNothing) Encode(parentID uint16, _ string, _ pcommon.Value) uint16 {
	return parentID
}

func (s *Attrs16ByNothing) Reset() {}

// Sorts the attributes by key, parentID, and value
// ================================================

func SortAttrs16ByKeyParentIdValue() *Attrs16ByKeyParentIdValue {
	return &Attrs16ByKeyParentIdValue{}
}

func (s *Attrs16ByKeyParentIdValue) Sort(attrs []Attr16) {
	sort.Slice(attrs, func(i, j int) bool {
		attrsI := attrs[i]
		attrsJ := attrs[j]
		if attrsI.Value.Type() == attrsJ.Value.Type() {
			if attrsI.Key == attrsJ.Key {
				if attrsI.ParentID == attrsJ.ParentID {
					return IsLess(attrsI.Value, attrsJ.Value)
				} else {
					return attrsI.ParentID < attrsJ.ParentID
				}
			} else {
				return attrsI.Key < attrsJ.Key
			}
		} else {
			return attrsI.Value.Type() < attrsJ.Value.Type()
		}
	})
}

func (s *Attrs16ByKeyParentIdValue) Encode(parentID uint16, key string, value pcommon.Value) uint16 {
	if s.prevValue == nil {
		s.prevKey = key
		s.prevValue = &value
		s.prevParentID = parentID
		return parentID
	}

	if s.IsSameGroup(key, value) {
		delta := parentID - s.prevParentID
		s.prevParentID = parentID
		return delta
	} else {
		s.prevKey = key
		s.prevValue = &value
		s.prevParentID = parentID
		return parentID
	}
}

func (s *Attrs16ByKeyParentIdValue) Reset() {
	s.prevParentID = 0
	s.prevKey = ""
	s.prevValue = nil
}

func (s *Attrs16ByKeyParentIdValue) IsSameGroup(key string, value pcommon.Value) bool {
	if s.prevValue == nil {
		return false
	}

	if s.prevValue.Type() == value.Type() && s.prevKey == key {
		return true
	}
	return false
}

// Sorts the attributes by key, value, and parentID
// ================================================

func SortAttrs16ByKeyValueParentId() *Attrs16ByKeyValueParentId {
	return &Attrs16ByKeyValueParentId{}
}

func (s *Attrs16ByKeyValueParentId) Sort(attrs []Attr16) {
	sort.Slice(attrs, func(i, j int) bool {
		attrsI := attrs[i]
		attrsJ := attrs[j]
		if attrsI.Value.Type() == attrsJ.Value.Type() {
			if attrsI.Key == attrsJ.Key {
				cmp := Compare(attrsI.Value, attrsJ.Value)
				if cmp == 0 {
					return attrsI.ParentID < attrsJ.ParentID
				} else {
					return cmp < 0
				}
			} else {
				return attrsI.Key < attrsJ.Key
			}
		} else {
			return attrsI.Value.Type() < attrsJ.Value.Type()
		}
	})
}

func (s *Attrs16ByKeyValueParentId) Encode(parentID uint16, key string, value pcommon.Value) uint16 {
	if s.prevValue == nil {
		s.prevKey = key
		s.prevValue = &value
		s.prevParentID = parentID
		return parentID
	}

	if s.IsSameGroup(key, value) {
		delta := parentID - s.prevParentID
		s.prevParentID = parentID
		return delta
	} else {
		s.prevKey = key
		s.prevValue = &value
		s.prevParentID = parentID
		return parentID
	}
}

func (s *Attrs16ByKeyValueParentId) Reset() {
	s.prevParentID = 0
	s.prevKey = ""
	s.prevValue = nil
}

func (s *Attrs16ByKeyValueParentId) IsSameGroup(key string, value pcommon.Value) bool {
	if s.prevValue == nil {
		return false
	}

	if s.prevKey == key {
		return Equal(*s.prevValue, value)
	} else {
		return false
	}
}
