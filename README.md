# cnb2cf
A tool to convert [Cloud Native Buildpacks](https://buildpacks.io/) (CNB's) to a single Cloudfoundry Buildpack

## Installation

### Requirements
- Go 1.11+

```
git clone https://github.com/cloudfoundry/cnb2cf
cd cnb2cf
scripts/build.sh
```

binary can be found in `build` dir

## Usage

```
$ cnb2cf shim.yml
```

Where `shim.yml` specifies the buildpacks, the order they will run in and some additional metadata. 

## Config

The config is a `.yml` file as the first argument and outputs a buildpack with the name `<language>_buildpack-<stack>-<version>.zip`

### Config file keys:

- `language` The language of the resulting CF Buildpack
- `stack` The CF stack the CNB's run on eg. `cflinuxfs3`
- `version` The version given to the resulting CF Buildpack
- `buildpacks` The CNB's to package together in the resulting CF Buildpack
  - `name` The CNB `id` found in the `order.toml`
  - `version` The CNB `version` found in the `order.toml`
  - `uri` A remote uri path to the cnb `tgz` archive eg. `S3`
  - `sha256` The sha256 of the `tgz` archive
- `groups` This is analogous to the `order.toml` in the [v3 buildpack spec](https://github.com/buildpack/spec/blob/master/platform.md) sans the CNB `version` which is not required

**Note**
`groups` defines how to run the CNB's so all CNB's in `groups` **must** exist in the `buildpacks` section

**Example shim.yml**

This example config creates a CF buildpack using the [python](https://github.com/cloudfoundry/python-cnb) and [pip](https://github.com/cloudfoundry/pip-cnb) Cloud Native Buildpacks

```
---
language: python
stack: cflinuxfs3
version: 1.0.0
buildpacks:
  - name: org.cloudfoundry.buildpacks.python
    version: 0.0.2
    uri: https://github.com/cloudfoundry/python-cnb/releases/download/v0.0.2/python-cnb-0.0.2.tgz
    sha256: 9a808c86c83c9bb42ec6e1ff3c005761b26ed4833f8f37bf282258839fc794a6
  - name: org.cloudfoundry.buildpacks.pip
    uri: https://github.com/cloudfoundry/pip-cnb/releases/download/v0.0.2/pip-cnb-0.0.2.tgz
    version: 0.0.2
    sha256: 37242c12d302cf1fdf85361fb5fcc845868c1f574308352c4c83a0ce99e9c99d
groups:
  - buildpacks:
      - id: "org.cloudfoundry.buildpacks.python"
      - id: "org.cloudfoundry.buildpacks.pip"
```

Results in a buildpack as a `.zip` file in the current directory which can be uploaded to Cloud Foundry

```
cf create-buildpack my_buildpack python_buildpack-cflinuxfs3-1.0.0.zip 10
```
