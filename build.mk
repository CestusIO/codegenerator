# -----------------------------------------------------------------------------
# DEFINES
# -----------------------------------------------------------------------------

# Directory to compile binaries to
BINDIR                  ?= bin
# List of platforms to target [linux/windows/darwin]
PLATFORMS               ?= linux
# List of architectures to target [amd64/arm64]
ARCHITECTURES           := amd64
# Name of the app used for single application builds
APP 					:= 
# List of applications to build (must reside in ./cmd/<name>)
APPLICATIONS            := 
# Buildtime of a version will be passed as ldflag to go compiler
VERSION_DATE            ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
# Default version
svermakerBuildVersion   ?= 
# GOPRIVATE will disable go cache
export GOPRIVATE        := code.cestus.io
# default docker version 
svermakerHelmLabel 	    ?= unreleased
goModuleBuildVersion    ?= unreleased
# additional LDFGLAGS (e.g. -w -s)
ADDITIONALLDFLAGS       ?= 