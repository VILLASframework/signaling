# SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.17-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./

RUN go build -o /server ./cmd/server/

FROM alpine

WORKDIR /

COPY --from=builder /server /

EXPOSE 8080

CMD ["/server"]
