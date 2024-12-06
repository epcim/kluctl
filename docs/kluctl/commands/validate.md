<!-- This comment is uncommented when auto-synced to www-kluctl.io

---
title: "validate"
linkTitle: "validate"
weight: 10
description: >
    validate command
---
-->

## Command
<!-- BEGIN SECTION "validate" "Usage" false -->
Usage: kluctl validate [flags]

Validates the already deployed deployment
This means that all objects are retrieved from the cluster and checked for readiness.

TODO: This needs to be better documented!

<!-- END SECTION -->

## Arguments
The following sets of arguments are available:
1. [project arguments](./common-arguments.md#project-arguments)
1. [image arguments](./common-arguments.md#image-arguments)
1. [helm arguments](./common-arguments.md#helm-arguments)
1. [registry arguments](./common-arguments.md#registry-arguments)

In addition, the following arguments are available:
<!-- BEGIN SECTION "validate" "Misc arguments" true -->
```
Misc arguments:
  Command specific arguments.

  -o, --output stringArray         Specify output target file. Can be specified multiple times
      --render-output-dir string   Specifies the target directory to render the project into. If omitted, a
                                   temporary directory is used.
      --sleep duration             Sleep duration between validation attempts (default 5s)
      --wait duration              Wait for the given amount of time until the deployment validates
      --warnings-as-errors         Consider warnings as failures

```
<!-- END SECTION -->
