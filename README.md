# cnb2cf
A tool to convert [Cloud Native Buildpacks](https://buildpacks.io/) (CNBs) to a single Cloud Foundry Buildpack

## Installation

### Requirements
- Go 1.11+

```
$ git clone https://github.com/cloudfoundry/cnb2cf
$ cd cnb2cf
$ ./scripts/build.sh
```

The binary (`cnb2cf`) can be found in the `build` dir.

## Usage
There are two cases when using this tool:

1. Create a shimmed buildpack `.zip` file from a `<config>.yml` configuration file (described below). **This is the recommmended path for most users.**  

      ```
      $ cnb2cf create -config <path to config file>.yml
      ```

1. Create a packaged shimmed buildpack `.zip` file from inside a shimmed buildpack's root directory. **This is only recommended if you are already shimming your buildpack to take care of specific edge cases.** 
      ```
      $ cnb2cf package -stack <stack> [-cached] [-version <version>] [-cachedir <path to cachedir>]
      ```      

_Note there is some overlap between these two use cases._

The output of both commands is a buildpack `.zip` file in the current directory, with the name `<language>_buildpack[-<cached>]-<stack>-<version>.zip`. That zip file can be then uploaded to Cloud Foundry by running
```
cf create-buildpack my_buildpack python_buildpack-cflinuxfs3-1.0.0.zip 10
```

## Config
The config is a `.yml` file which specifies the buildpacks, the order they will run in, and some additional metadata. 

### Config keys:
- `language` &rarr; The language of the resulting CF Buildpack
- `stack` &rarr; The CF stack the CNBs run on eg. `cflinuxfs3`
- `version` &rarr; The version given to the resulting CF Buildpack
- `buildpacks` &rarr; The CNBs to package together in the resulting CF Buildpack
  - `name` &rarr; The CNB `id` found in its `order.toml`
  - `version` &rarr; The CNB `version` found in its `order.toml`
  - `uri` &rarr; A remote uri path to the cnb `tgz` archive eg. `S3`
  - `sha256` &rarr; The sha256 of the `tgz` archive
- `groups` &rarr; This is analogous to the `order.toml` in the [v3 buildpack spec](https://github.com/buildpack/spec/blob/master/platform.md) sans the CNB `version` which is not required

**Note**

`groups` defines how to run the CNBs, so all CNBs in `groups` **must** exist in the `buildpacks` section

### Example <config>.yml
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

### Simple Workflow Example

A simple example workflow using the config file above named `shim.yml`:
```
$ cnb2cf create -config shim.yml
# the above produces python_buildpack-cflinuxfs3-1.0.0.zip

# then upload to Cloud Foundry using the cf cli
$ cf create-buildpack my_shimmed_buildpack python_buildpack-cflinuxfs3-1.0.0.zip 1
```
