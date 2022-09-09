package ddfasthttp

import (
	"context"
	"net/http"
	"os"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// DataDogTransaction - key context transaction
const DataDogTransaction = "__datadog_ctx_fasthttp__"

// FromContext - datadog from context
func FromContext(ctx *fasthttp.RequestCtx) *ddtrace.Span {
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
	return &span
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
