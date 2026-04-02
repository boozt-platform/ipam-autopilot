# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

FROM golang:1.26-bookworm AS build

WORKDIR /go/src/app
COPY . .

RUN CGO_ENABLED=0 go build -o /go/bin/ipam-autopilot .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /go/bin/ipam-autopilot /
USER nonroot:nonroot
CMD ["/ipam-autopilot"]
