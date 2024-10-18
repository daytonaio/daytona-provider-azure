<div align="center">

[![License](https://img.shields.io/badge/License-MIT-blue)](#license)
[![Go Report Card](https://goreportcard.com/badge/github.com/daytonaio/daytona-provider-azure)](https://goreportcard.com/report/github.com/daytonaio/daytona-provider-azure)
[![Issues - daytona](https://img.shields.io/github/issues/daytonaio/daytona-azure-provider)](https://github.com/daytonaio/daytona-provider-azure/issues)
![GitHub Release](https://img.shields.io/github/v/release/daytonaio/daytona-azure-provider)

</div>

<h1 align="center">Daytona Azure Provider</h1>
<div align="center">
This repository is the home of the <a href="https://github.com/daytonaio/daytona">Daytona</a> Azure Provider.
</div>
</br>

<p align="center">
  <a href="https://github.com/daytonaio/daytona-provider-azure/issues/new?assignees=&labels=bug&projects=&template=bug_report.md&title=%F0%9F%90%9B+Bug+Report%3A+">Report Bug</a>
    ·
  <a href="https://github.com/daytonaio/daytona-provider-azure/issues/new?assignees=&labels=enhancement&projects=&template=feature_request.md&title=%F0%9F%9A%80+Feature%3A+">Request Feature</a>
    ·
  <a href="https://go.daytona.io/slack">Join Our Slack</a>
    ·
  <a href="https://x.com/Daytonaio">X</a>
</p>

The Azure Provider allows Daytona to create and manage workspace projects on Amazon virtual machines.

To use the Azure Provider for managing workspace projects, you'll need ensure that the token is associated with a
service principal with the `Contributor` role assigned to the relevant Azure subscription.
You can create a service principal with the Contributor role using the following Azure CLI command:

```shell
az ad sp create-for-rbac --role Contributor --scopes /subscriptions/<subscription-id>
```

Detailed instructions on generating and configuring the token can be found [here](https://learn.microsoft.com/en-us/cli/azure/azure-cli-sp-tutorial-1?tabs=bash)

## Target Options

| Property        | Type   | Optional | DefaultValue                             | InputMasked | DisabledPredicate |
| --------------- | ------ | -------- | ---------------------------------------- | ----------- | ----------------- |
| Region          | String | true     | centralus                                | false       |                   |
| Image URN       | String | true     | Canonical:ubuntu-24_04-lts:server:latest | false       |                   |
| VM Size         | String | true     | Standard_B2s                             | false       |                   |
| Disk Type       | String | true     | StandardSSD_LRS                          | false       |                   |
| Disk Size       | Int    | true     | 30                                       | false       |                   |
| Resource Group  | String | true     |                                          | false       |                   |
| Tenant Id       | String | false    |                                          | true        |                   |
| Client Id       | String | false    |                                          | true        |                   |
| Client Secret   | String | false    |                                          | true        |                   |
| Subscription Id | String | false    |                                          | true        |                   |

### Preset Targets

The Azure Provider has no preset targets. Before using the provider you must set the target using the daytona target set command.

## Code of Conduct

This project has adapted the Code of Conduct from the [Contributor Covenant](https://www.contributor-covenant.org/). For more information see the [Code of Conduct](CODE_OF_CONDUCT.md) or contact [codeofconduct@daytona.io.](mailto:codeofconduct@daytona.io) with any additional questions or comments.

## Contributing

The Daytona Docker Provider is Open Source under the [MIT License](LICENSE). If you would like to contribute to the software, you must:

1. Read the Developer Certificate of Origin Version 1.1 (https://developercertificate.org/)
2. Sign all commits to the Daytona Docker Provider project.

This ensures that users, distributors, and other contributors can rely on all the software related to Daytona being contributed under the terms of the [License](LICENSE). No contributions will be accepted without following this process.

Afterwards, navigate to the [contributing guide](CONTRIBUTING.md) to get started.

## Questions

For more information on how to use and develop Daytona, talk to us on
[Slack](https://go.daytona.io/slack).
