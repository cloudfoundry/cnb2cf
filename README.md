# cnb2cf
A tool to convert [Cloud Native Buildpacks](https://buildpacks.io/) (CNBs) to a single Cloud Foundry Buildpack (Shimmed buildpack)

A _shimmed buildpack_ is a wrapper around a group of CNBs, allowing them to run on Cloud Foundry deployments.

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

1. Create a shimmed buildpack `.zip` file from within a shimmed buildpacks root directory. This requires more specific setup, but allows you to cache the CNB dependencies in your shimmed buildpack, and to be run as a github url. **This is only recommended if you are already shimming your buildpack to take care of specific edge cases.**
      ```
      $ cnb2cf package -stack <stack> [-cached] [-version <version>] [-cachedir <path to cachedir>]
      ```

1. Create a shimmed buildpack `.zip` file from a `<config>.yml` configuration file (described below). That buildpack can then be run alone, or integrated with other buildpacks (using [multi buildpack](https://docs.cloudfoundry.org/buildpacks/use-multiple-buildpacks.html)). **This is the recommmended path for most users.**

      ```
      $ cnb2cf create -config <path to config file>.yml
      ```

_Note there is some overlap between these two use cases._

The output of both commands is a buildpack `.zip` file in the current directory, with the name `<language>_buildpack[-<cached>]-<stack>-<version>.zip`. That zip file can be then uploaded to Cloud Foundry by running
```
cf create-buildpack my_buildpack python_buildpack-cflinuxfs3-1.0.0.zip 10
```

# Package
`cnb2cf package -stack <stack> [-cached] [-version <version>] [-cachedir <path to cachedir>]`

This command must be run in the directory of a shimmed buildpack repo. An example shimmed buildpack repo can be found [here](https://github.com/cloudfoundry/nodejs-buildpack/tree/v3). 

### Shimmed Buildpack Repo Setup:
- `manifest.yml` &rarr; A Cloud Foundry buildpack manifest with the dependencies needed to run as a shim:
  - `lifecycle` &rarr; The binary required to run the CNB lifecycle, per the [Buildpack Spec](https://github.com/buildpack/spec/blob/master/platform.md) 
  - CNBs &rarr; The CNBs used in the group for this shimmed buildpack. Each of these (eg. the [nodejs-cnb](https://www.github.com/cloudfoundry/nodejs-cnb)) require the following keys:
    - `name` &rarr; The CNB `id` found in its `order.toml`. Eg: `org.cloudfoundry.buildpacks.nodejs`
    - `version` &rarr; The CNB `version` found in its `order.toml`. Eg: `0.0.8`
    - `uri` &rarr; A remote uri path to the CNB `tgz` archive. Eg: `https://buildpacks.cloudfoundry.org/dependencies/org.cloudfoundry.buildpacks.nodejs/org.cloudfoundry.buildpacks.nodejs-0.0.8-any-stack-b75d0983.tgz`
    - `sha256` &rarr; The sha256 of the `tgz` archive. Eg: `b75d0983831fc10e55076d7c2a7aa52b0131964c48eaa8de5d19d30a3e2b1abb`
    - `cf_stacks`: The stacks upon which it works. Eg: `cflinuxfs3`
    - `source` &rarr; The source code of the CNB. Eg: `https://github.com/cloudfoundry/nodejs-cnb/archive/v0.0.8.tar.gz`
    - `source_sha256` &rarr; The sha256 of the source archive. Eg: `6d1a5d98792acc2f07705abec5a63674577c5659aea7e3621dc20ea8d10456cd`
  - `pre_package` script &rarr; This pre_package script will be used to build the binaries, in order to run the shim. Eg: `scripts/build`
- `order.toml` &rarr; The groups and order definition for the CNBs, per the [Buildpack Spec](https://github.com/buildpack/spec/blob/master/platform.md#ordertoml-toml)
- `bin` directory &rarr; The set of Cloud Foundry Buildpack executables required to run CF lifecycle per [CF docs](https://docs.cloudfoundry.org/buildpacks/understand-buildpacks.html#buildpack-scripts)
- `scripts/build.sh` &rarr; The script, pointed to in the `pre_package` key of the manifest, which tells the buildpack lifecycle how to build the binaries. You can copy the script from [nodejs-buildpack#v3](https://github.com/cloudfoundry/nodejs-buildpack/blob/v3/scripts/build.sh) 
 
# Create
`cnb2cf create -config <path to config file>.yml`

## Config File
The config is a `.yml` file which specifies the buildpacks, the order they will run in, and some additional metadata. 

### Config keys:
- `language` &rarr; The language of the resulting CF Buildpack
- `stack` &rarr; The CF stack the CNBs run on eg. `cflinuxfs3`
- `version` &rarr; The version given to the resulting CF Buildpack
- `buildpacks` &rarr; The CNBs to package together in the resulting CF Buildpack
  - `name` &rarr; The CNB `id` found in its `order.toml`
  - `version` &rarr; The CNB `version` found in its `order.toml`
  - `uri` &rarr; A remote uri path to the CNB `tgz` archive eg. `S3`
  - `sha256` &rarr; The sha256 of the `tgz` archive
- `groups` &rarr; This is analogous to the `order.toml` in the [v3 buildpack spec](https://github.com/buildpack/spec/blob/master/platform.md) sans the CNB `version` which is not required

**Note**

`groups` defines how to run the CNBs, so all CNBs in `groups` **must** exist in the `buildpacks` section

### Example \<config\>.yml
This example config creates a CF buildpack using the [python](https://github.com/cloudfoundry/python-cnb) and [pip](https://github.com/cloudfoundry/pip-cnb) Cloud Native Buildpacks

```
---
language: python
stack: cflinuxfs3
version: 1.0.0
buildpacks:
  - name: org.cloudfoundry.buildpacks.python
    version: 0.0.4
    uri: https://github.com/cloudfoundry/python-cnb/releases/download/v0.0.4/python-cnb-0.0.4.tgz
    sha256: eeac90a03ab5f0fa0e287125b2b0744938340a2409ff3d5cdfcf77294282f474
  - name: org.cloudfoundry.buildpacks.pip
    uri: https://github.com/cloudfoundry/pip-cnb/releases/download/v0.0.5/pip-cnb-0.0.5.tgz
    version: 0.0.5
    sha256: c6b5c7ec13d5e484fbc6e3ffcc760460eb2279a9ad768320bb7d95d7d33367ef
groups:
  - buildpacks:
      - id: "org.cloudfoundry.buildpacks.python"
      - id: "org.cloudfoundry.buildpacks.pip"
```

## Simple Workflow Example

A simple example workflow using the config file above named `shim.yml`:
```
$ cnb2cf create -config shim.yml
# the above produces python_buildpack-cflinuxfs3-1.0.0.zip

# then upload to Cloud Foundry using the cf cli
$ cf create-buildpack my_shimmed_buildpack python_buildpack-cflinuxfs3-1.0.0.zip 1
```
