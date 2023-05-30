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

import (
	"github.com/apache/arrow/go/v12/arrow"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/f5/otel-arrow-adapter/pkg/otel/common"
	acommon "github.com/f5/otel-arrow-adapter/pkg/otel/common/arrow"
	"github.com/f5/otel-arrow-adapter/pkg/otel/common/schema"
	"github.com/f5/otel-arrow-adapter/pkg/otel/common/schema/builder"
	"github.com/f5/otel-arrow-adapter/pkg/otel/constants"
	"github.com/f5/otel-arrow-adapter/pkg/otel/stats"
	"github.com/f5/otel-arrow-adapter/pkg/werror"
)

var (
	// LogsSchema is the Arrow schema for the OTLP Arrow Logs record.
	LogsSchema = arrow.NewSchema([]arrow.Field{
		{Name: constants.ID, Type: arrow.PrimitiveTypes.Uint16, Metadata: schema.Metadata(schema.Optional, schema.DeltaEncoding)},
		{Name: constants.Resource, Type: acommon.ResourceDT, Metadata: schema.Metadata(schema.Optional)},
		{Name: constants.Scope, Type: acommon.ScopeDT, Metadata: schema.Metadata(schema.Optional)},
		// This schema URL applies to the span and span events (the schema URL
		// for the resource is in the resource struct).
		{Name: constants.SchemaUrl, Type: arrow.BinaryTypes.String, Metadata: schema.Metadata(schema.Optional, schema.Dictionary8)},
		{Name: constants.TimeUnixNano, Type: arrow.FixedWidthTypes.Timestamp_ns},
		{Name: constants.ObservedTimeUnixNano, Type: arrow.FixedWidthTypes.Timestamp_ns},
		{Name: constants.TraceId, Type: &arrow.FixedSizeBinaryType{ByteWidth: 16}, Metadata: schema.Metadata(schema.Optional, schema.Dictionary8)},
		{Name: constants.SpanId, Type: &arrow.FixedSizeBinaryType{ByteWidth: 8}, Metadata: schema.Metadata(schema.Optional, schema.Dictionary8)},
		{Name: constants.SeverityNumber, Type: arrow.PrimitiveTypes.Int32, Metadata: schema.Metadata(schema.Optional, schema.Dictionary8)},
		{Name: constants.SeverityText, Type: arrow.BinaryTypes.String, Metadata: schema.Metadata(schema.Optional, schema.Dictionary8)},
		{Name: constants.Body, Type: arrow.StructOf([]arrow.Field{
			{Name: constants.BodyType, Type: arrow.PrimitiveTypes.Uint8},
			{Name: constants.BodyStr, Type: arrow.BinaryTypes.String, Metadata: schema.Metadata(schema.Dictionary16)},
			{Name: constants.BodyInt, Type: arrow.PrimitiveTypes.Int64, Metadata: schema.Metadata(schema.Optional, schema.Dictionary16)},
			{Name: constants.BodyDouble, Type: arrow.PrimitiveTypes.Float64, Metadata: schema.Metadata(schema.Optional)},
			{Name: constants.BodyBool, Type: arrow.FixedWidthTypes.Boolean, Metadata: schema.Metadata(schema.Optional)},
			{Name: constants.BodyBytes, Type: arrow.BinaryTypes.Binary, Metadata: schema.Metadata(schema.Optional, schema.Dictionary16)},
			{Name: constants.BodySer, Type: arrow.BinaryTypes.Binary, Metadata: schema.Metadata(schema.Optional, schema.Dictionary16)},
		}...), Metadata: schema.Metadata(schema.Optional)},
		{Name: constants.DroppedAttributesCount, Type: arrow.PrimitiveTypes.Uint32, Metadata: schema.Metadata(schema.Optional)},
		{Name: constants.Flags, Type: arrow.PrimitiveTypes.Uint32, Metadata: schema.Metadata(schema.Optional)},
	}, nil)
)

// LogsBuilder is a helper to build a list of resource logs.
type LogsBuilder struct {
	released bool

	builder *builder.RecordBuilderExt // Record builder

	rb    *acommon.ResourceBuilder        // `resource` builder
	scb   *acommon.ScopeBuilder           // `scope` builder
	sschb *builder.StringBuilder          // scope `schema_url` builder
	ib    *builder.Uint16DeltaBuilder     //  id builder
	tub   *builder.TimestampBuilder       // `time_unix_nano` builder
	otub  *builder.TimestampBuilder       // `observed_time_unix_nano` builder
	tidb  *builder.FixedSizeBinaryBuilder // `trace_id` builder
	sidb  *builder.FixedSizeBinaryBuilder // `span_id` builder
	snb   *builder.Int32Builder           // `severity_number` builder
	stb   *builder.StringBuilder          // `severity_text` builder

	bodyb *builder.StructBuilder // `body` builder
	typeb *builder.Uint8Builder
	strb  *builder.StringBuilder
	i64b  *builder.Int64Builder
	f64b  *builder.Float64Builder
	boolb *builder.BooleanBuilder
	binb  *builder.BinaryBuilder
	serb  *builder.BinaryBuilder

	dacb *builder.Uint32Builder // `dropped_attributes_count` builder
	fb   *builder.Uint32Builder // `flags` builder

	optimizer *LogsOptimizer
	analyzer  *LogsAnalyzer

	relatedData *RelatedData
}

// NewLogsBuilder creates a new LogsBuilder with a given allocator.
func NewLogsBuilder(
	recordBuilder *builder.RecordBuilderExt,
	cfg *Config,
	stats *stats.ProducerStats,
) (*LogsBuilder, error) {
	var optimizer *LogsOptimizer
	var analyzer *LogsAnalyzer

	relatedData, err := NewRelatedData(cfg, stats)
	if err != nil {
		panic(err)
	}

	if stats.SchemaStatsEnabled {
		optimizer = NewLogsOptimizer(cfg.Log.Sorter)
		analyzer = NewLogsAnalyzer()
	} else {
		optimizer = NewLogsOptimizer(cfg.Log.Sorter)
	}

	b := &LogsBuilder{
		released:    false,
		builder:     recordBuilder,
		optimizer:   optimizer,
		analyzer:    analyzer,
		relatedData: relatedData,
	}

	if err := b.init(); err != nil {
		return nil, werror.Wrap(err)
	}

	return b, nil
}

func (b *LogsBuilder) init() error {
	ib := b.builder.Uint16DeltaBuilder(constants.ID)
	// As the attributes are sorted before insertion, the delta between two
	// consecutive attributes ID should always be <=1.
	ib.SetMaxDelta(1)

	b.ib = ib
	b.rb = acommon.ResourceBuilderFrom(b.builder.StructBuilder(constants.Resource))
	b.scb = acommon.ScopeBuilderFrom(b.builder.StructBuilder(constants.Scope))
	b.sschb = b.builder.StringBuilder(constants.SchemaUrl)

	b.tub = b.builder.TimestampBuilder(constants.TimeUnixNano)
	b.otub = b.builder.TimestampBuilder(constants.ObservedTimeUnixNano)
	b.tidb = b.builder.FixedSizeBinaryBuilder(constants.TraceId)
	b.sidb = b.builder.FixedSizeBinaryBuilder(constants.SpanId)
	b.snb = b.builder.Int32Builder(constants.SeverityNumber)
	b.stb = b.builder.StringBuilder(constants.SeverityText)

	b.bodyb = b.builder.StructBuilder(constants.Body)
	b.typeb = b.bodyb.Uint8Builder(constants.BodyType)
	b.strb = b.bodyb.StringBuilder(constants.BodyStr)
	b.i64b = b.bodyb.Int64Builder(constants.BodyInt)
	b.f64b = b.bodyb.Float64Builder(constants.BodyDouble)
	b.boolb = b.bodyb.BooleanBuilder(constants.BodyBool)
	b.binb = b.bodyb.BinaryBuilder(constants.BodyBytes)
	b.serb = b.bodyb.BinaryBuilder(constants.BodySer)

	b.dacb = b.builder.Uint32Builder(constants.DroppedAttributesCount)
	b.fb = b.builder.Uint32Builder(constants.Flags)

	return nil
}

func (b *LogsBuilder) RelatedData() *RelatedData {
	return b.relatedData
}

// Build builds an Arrow Record from the builder.
//
// Once the array is no longer needed, Release() must be called to free the
// memory allocated by the record.
func (b *LogsBuilder) Build() (record arrow.Record, err error) {
	if b.released {
		return nil, werror.Wrap(acommon.ErrBuilderAlreadyReleased)
	}

	record, err = b.builder.NewRecord()
	if err != nil {
		initErr := b.init()
		if initErr != nil {
			err = werror.Wrap(initErr)
		}
	}

	return
}

// Append appends a new set of resource logs to the builder.
func (b *LogsBuilder) Append(logs plog.Logs) (err error) {
	if b.released {
		return werror.Wrap(acommon.ErrBuilderAlreadyReleased)
	}

	optimLogs := b.optimizer.Optimize(logs)
	if b.analyzer != nil {
		b.analyzer.Analyze(optimLogs)
		b.analyzer.ShowStats("")
	}

	attrsAccu := b.relatedData.AttrsBuilders().LogRecord().Accumulator()

	logID := uint16(0)
	var resLogID, scopeLogID string
	var resID, scopeID int64

	for _, logRec := range optimLogs.Logs {
		log := logRec.Log
		logAttrs := log.Attributes()

		ID := logID

		if logAttrs.Len() == 0 {
			b.ib.AppendNull()
		} else {
			b.ib.Append(ID)
			logID++
		}

		// Resource spans
		if resLogID != logRec.ResourceLogsID {
			resLogID = logRec.ResourceLogsID
			resID, err = b.relatedData.AttrsBuilders().Resource().Accumulator().Append(logRec.Resource.Attributes())
			if err != nil {
				return werror.Wrap(err)
			}
		}
		if err = b.rb.AppendWithID(resID, logRec.Resource, logRec.ResourceSchemaUrl); err != nil {
			return werror.Wrap(err)
		}

		// Scope spans
		if scopeLogID != logRec.ScopeLogsID {
			scopeLogID = logRec.ScopeLogsID
			scopeID, err = b.relatedData.AttrsBuilders().scope.Accumulator().Append(logRec.Scope.Attributes())
			if err != nil {
				return werror.Wrap(err)
			}
		}
		if err = b.scb.AppendWithAttrsID(scopeID, logRec.Scope); err != nil {
			return werror.Wrap(err)
		}
		b.sschb.AppendNonEmpty(logRec.ScopeSchemaUrl)

		b.tub.Append(arrow.Timestamp(log.Timestamp()))
		b.otub.Append(arrow.Timestamp(log.ObservedTimestamp()))
		tib := log.TraceID()
		b.tidb.Append(tib[:])
		sib := log.SpanID()
		b.sidb.Append(sib[:])
		b.snb.AppendNonZero(int32(log.SeverityNumber()))
		b.stb.AppendNonEmpty(log.SeverityText())

		// Log record body
		body := log.Body()
		switch body.Type() {
		case pcommon.ValueTypeStr:
			err = b.bodyb.Append(body, func() error {
				b.typeb.Append(uint8(pcommon.ValueTypeStr))
				b.strb.Append(body.Str())
				b.i64b.AppendNull()
				b.f64b.AppendNull()
				b.boolb.AppendNull()
				b.binb.AppendNull()
				b.serb.AppendNull()
				return nil
			})
			if err != nil {
				return werror.Wrap(err)
			}
		case pcommon.ValueTypeInt:
			err = b.bodyb.Append(body, func() error {
				b.typeb.Append(uint8(pcommon.ValueTypeInt))
				b.i64b.Append(body.Int())
				b.strb.AppendNull()
				b.f64b.AppendNull()
				b.boolb.AppendNull()
				b.binb.AppendNull()
				b.serb.AppendNull()
				return nil
			})
			if err != nil {
				return werror.Wrap(err)
			}
		case pcommon.ValueTypeDouble:
			err = b.bodyb.Append(body, func() error {
				b.typeb.Append(uint8(pcommon.ValueTypeDouble))
				b.f64b.Append(body.Double())
				b.strb.AppendNull()
				b.i64b.AppendNull()
				b.boolb.AppendNull()
				b.binb.AppendNull()
				b.serb.AppendNull()
				return nil
			})
			if err != nil {
				return werror.Wrap(err)
			}
		case pcommon.ValueTypeBool:
			err = b.bodyb.Append(body, func() error {
				b.typeb.Append(uint8(pcommon.ValueTypeBool))
				b.boolb.Append(body.Bool())
				b.strb.AppendNull()
				b.i64b.AppendNull()
				b.f64b.AppendNull()
				b.binb.AppendNull()
				b.serb.AppendNull()
				return nil
			})
			if err != nil {
				return werror.Wrap(err)
			}
		case pcommon.ValueTypeBytes:
			err = b.bodyb.Append(body, func() error {
				b.typeb.Append(uint8(pcommon.ValueTypeBytes))
				b.binb.Append(body.Bytes().AsRaw())
				b.strb.AppendNull()
				b.i64b.AppendNull()
				b.f64b.AppendNull()
				b.boolb.AppendNull()
				b.serb.AppendNull()
				return nil
			})
			if err != nil {
				return werror.Wrap(err)
			}
		case pcommon.ValueTypeSlice:
			cborData, err := common.Serialize(body)
			if err != nil {
				return werror.Wrap(err)
			}
			err = b.bodyb.Append(body, func() error {
				b.typeb.Append(uint8(pcommon.ValueTypeSlice))
				b.serb.Append(cborData)
				b.strb.AppendNull()
				b.i64b.AppendNull()
				b.f64b.AppendNull()
				b.boolb.AppendNull()
				b.binb.AppendNull()
				return nil
			})
			if err != nil {
				return werror.Wrap(err)
			}
		case pcommon.ValueTypeMap:
			cborData, err := common.Serialize(body)
			if err != nil {
				return werror.Wrap(err)
			}
			err = b.bodyb.Append(body, func() error {
				b.typeb.Append(uint8(pcommon.ValueTypeMap))
				b.serb.Append(cborData)
				b.strb.AppendNull()
				b.i64b.AppendNull()
				b.f64b.AppendNull()
				b.boolb.AppendNull()
				b.binb.AppendNull()
				return nil
			})
			if err != nil {
				return werror.Wrap(err)
			}
		case pcommon.ValueTypeEmpty:
			b.bodyb.AppendNull()
		}

		// Log record attributes
		if logAttrs.Len() > 0 {
			err := attrsAccu.AppendWithID(ID, log.Attributes())
			if err != nil {
				return werror.Wrap(err)
			}
		}

		b.dacb.AppendNonZero(log.DroppedAttributesCount())

		b.fb.Append(uint32(log.Flags()))
	}
	return nil
}

// Release releases the memory allocated by the builder.
func (b *LogsBuilder) Release() {
	if !b.released {
		b.builder.Release()
		b.released = true

		b.relatedData.Release()
	}
}

func (b *LogsBuilder) ShowSchema() {
	b.builder.ShowSchema()
}
