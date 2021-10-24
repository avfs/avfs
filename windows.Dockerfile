##
##	Copyright 2021 The AVFS authors
##
##	Licensed under the Apache License, Version 2.0 (the "License");
##	you may not use this file except in compliance with the License.
##	You may obtain a copy of the License at
##
##		http://www.apache.org/licenses/LICENSE-2.0
##
##	Unless required by applicable law or agreed to in writing, software
##	distributed under the License is distributed on an "AS IS" BASIS,
##	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
##	See the License for the specific language governing permissions and
##	limitations under the License.
##

FROM golang:windowsservercore AS base
USER ContainerAdministrator
WORKDIR /go/src

FROM base AS avfs
COPY mage mage
RUN go run mage/build.go

FROM base AS modules
COPY go.mod go.sum ./
RUN go mod download

# This should fail xith error go: warning: "./..." matched no packages
# each time the modules are changed, but it avoids downloading
# modules for each build.
RUN go build ./...

FROM base AS copyfiles
COPY --from=avfs /go/bin /go/bin
COPY --from=modules /go/pkg /go/pkg
COPY ./go.mod ./go.sum ./
COPY *.go ./
COPY idm idm
COPY test test
COPY vfs vfs

FROM copyfiles
CMD avfs test
