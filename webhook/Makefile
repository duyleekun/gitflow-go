version ?= 0.0.0
module=webhook
outfile=./bin/$(module)-$(1)-$(2)-$(version)
test_opts?=

define build_arch_os
	GOARCH=$(1) GOOS=$(2) go build -o "$(outfile)"
	chmod a+x "$(outfile)"
endef

define build_arch_os_and_run
	GOARCH=$(1) GOOS=$(2) go build -o "$(outfile)"
	chmod a+x "$(outfile)"
	"$(outfile)" ${test_opts}
endef

build: build_linux build_darwin build_arm

build_arm:
	$(call build_arch_os,arm,linux)
build_linux:
	$(call build_arch_os,amd64,linux)
build_darwin:
	$(call build_arch_os,amd64,darwin)
test_darwin: build_darwin
	$(call build_arch_os_and_run,amd64,darwin)
release: build