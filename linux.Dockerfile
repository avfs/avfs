##
##	Copyright 2020 The AVFS authors
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

FROM golang:buster AS base
WORKDIR /gopath/src

FROM base AS copyfiles
COPY ./bin/avfs /gopath/bin/
COPY ./*.go ./
COPY ./go.mod ./
COPY ./idm ./idm
COPY ./test ./test
COPY ./vfs ./vfs
COPY ./vfsutils ./vfsutils

FROM copyfiles
ENV PATH "/gopath/bin:$PATH"
CMD avfs test
