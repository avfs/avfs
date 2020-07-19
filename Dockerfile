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

# go test -race needs CGO_ENABLED=1, gcc and glibc
# Alpine Linux does not provide glibc and can't be used https://github.com/golang/go/issues/14481
FROM golang:buster
WORKDIR /go/src
COPY . .
RUN ln ./mage/bin/avfs ../bin/avfs
CMD avfs test
