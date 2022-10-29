package ddfasthttp

import (
	"context"
	"net/http"
	"os"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// DataDogTransaction - key context transaction
const DataDogTransaction = "__datadog_ctx_fasthttp__"

// SpanTags is tags of span.
type SpanTags map[string]interface{}

// DDTraceResult provides result.
type DDTraceResult func() (SpanTags, error)

// FromContext - datadog from context
func FromContext(ctx *fasthttp.RequestCtx) ddtrace.Span {
	val := ctx.UserValue(DataDogTransaction)
	if val == nil {
		return nil
	}
	element, ok := val.(context.Context)
	if !ok {
		return nil
	}
	span, ok := tracer.SpanFromContext(element)
	if !ok {
		return nil
	}
	return span
}

// Middleware - middleware para fasthpp new relic
func Middleware(f fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {

		if _, ok := os.LookupEnv("DATADOG_ENABLED"); !ok {
			f(ctx)
			return
		}

		spanOpts := []ddtrace.StartSpanOption{
			tracer.ResourceName(string(ctx.Request.Header.Method()) + " " + string(ctx.Request.URI().Path())),
		}

		var r http.Request
		if e := fasthttpadaptor.ConvertRequest(ctx, &r, true); e != nil {
			panic(e)
		}

		span, context := StartRequestSpan(&r, spanOpts...)

		ctx.SetUserValue(DataDogTransaction, context)

		f(ctx)

		statusCode := ctx.Response.StatusCode()

		FinishRequestSpan(span, statusCode)
	}
}

// StartChildSpan child span
func StartChildSpan(ctx *fasthttp.RequestCtx, operationName string, tags SpanTags) ddtrace.Span {
	txn := FromContext(ctx)
	if txn == nil {
		return nil
	}
	return StartDDSpan(operationName, txn, "", tags)
}

// StartDDSpan starts a datadog span.
func StartDDSpan(operationName string, parentSpan tracer.Span, spanType string, tags SpanTags) tracer.Span {
	var span tracer.Span
	if parentSpan != nil {
		span = tracer.StartSpan(operationName, tracer.ChildOf(parentSpan.Context()))
	} else {
		span = tracer.StartSpan(operationName)
	}
	if len(spanType) > 0 {
		tags[ext.SpanType] = spanType
	}
	setSpanTags(span, tags)
	return span
}

// EndSpan finishes a datadog span.
func EndSpan(span tracer.Span) {
	if span != nil {
		span.Finish()
	}
}

// EndSpanError finishes a datadog span.
func EndSpanError(span tracer.Span, e error) {
	if span != nil && e != nil {
		span.Finish(tracer.WithError(e))
	}
	if span != nil && e == nil {
		span.Finish()
	}
}

// EndSpanTags finishes a datadog span.
func EndSpanTags(span tracer.Span, tags SpanTags) {
	setSpanTags(span, tags)
	if span != nil {
		span.Finish()
	}
}

// EndSpanTagsError finishes a datadog span.
func EndSpanTagsError(span tracer.Span, tags SpanTags, e error) {
	setSpanTags(span, tags)
	if span != nil && e != nil {
		span.Finish(tracer.WithError(e))
	}
	if span != nil && e == nil {
		span.Finish()
	}
}

func setSpanTags(span tracer.Span, tags SpanTags) {
	if len(tags) > 0 {
		for k, v := range tags {
			span.SetTag(k, v)
		}
	}
}
