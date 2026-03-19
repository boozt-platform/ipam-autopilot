// Copyright 2026 Boozt Fashion AB
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

package main

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

func requestIDMiddleware() fiber.Handler {
	return requestid.New()
}

func accessLogMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		slog.Info("request",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", c.GetRespHeader(fiber.HeaderXRequestID),
		)
		return err
	}
}

func tracingMiddleware() fiber.Handler {
	tracer := otel.Tracer("ipam-autopilot")
	propagator := otel.GetTextMapPropagator()

	return func(c *fiber.Ctx) error {
		headers := make(propagation.MapCarrier)
		c.Request().Header.VisitAll(func(key, val []byte) {
			headers[string(key)] = string(val)
		})
		ctx := propagator.Extract(c.Context(), headers)
		spanName := c.Method() + " " + c.Path()
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", c.Method()),
			attribute.String("http.url", c.OriginalURL()),
			attribute.String("http.request_id", c.GetRespHeader(fiber.HeaderXRequestID)),
		)

		c.SetUserContext(ctx)
		err := c.Next()

		span.SetAttributes(attribute.Int("http.status_code", c.Response().StatusCode()))
		return err
	}
}
