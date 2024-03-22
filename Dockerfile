FROM golang:1.22-alpine AS build
ARG APP_VERSION=development

# Build outside of $GOPATH, it's simpler when using Go Modules.
WORKDIR /src

# If you need vendoring you can uncomment this
#COPY vendor ./vendor
COPY go.mod go.sum ./

RUN apk add --update make git; go mod download

# Copy everything else.
COPY . .

# Build a static binary.
RUN CGO_ENABLED=0 GOPATH=/src/.go make build && mv ./dist /go/bin/app
# Verify if the binary is truly static.
RUN ldd /go/bin/app 2>&1 | grep -q 'Not a valid dynamic program'

FROM alpine
COPY --from=build /go/bin/app /usr/local/bin

CMD ["transactions"]
