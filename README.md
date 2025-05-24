# README

Note: 该项目 fork 自 go-delve/delve，本项目做了一些功能删减，以使其适合作为电子书《go 符号级调试器设计开发》的示例，方便读者朋友能更方便快速地了解调试器设计开发的核心知识点。

...

2025/05/24，这个 delve 版本太久了(3.5 年前的版本），最多支持到 go1.17。但是我希望读者朋友使用较新的 go 版本 1.24 也可以正常使用。
所以这个项目要先暂时 archieve 一下。然后我重新从最新 delve 最新版拉一个分支出来。

---

![Delve](https://raw.githubusercontent.com/go-delve/delve/master/assets/delve_horizontal.png)

[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/go-delve/delve/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/go-delve/delve?status.svg)](https://godoc.org/github.com/go-delve/delve)
[![Build Status](<https://delve.beta.teamcity.com/app/rest/builds/buildType:(id:Delve_AggregatorBuild)/statusIcon.svg>)](https://delve.beta.teamcity.com/viewType.html?buildTypeId=Delve_AggregatorBuild&guest=1)

The GitHub issue tracker is for **bugs** only. Please use the [developer mailing list](https://groups.google.com/forum/#!forum/delve-dev) for any feature proposals and discussions.

### About Delve

- [Installation](Documentation/installation)
- [Getting Started](Documentation/cli/getting_started.md)
- [Documentation](Documentation)
  - [Command line options](Documentation/usage/dlv.md)
  - [Command line client](Documentation/cli/README.md)
  - [Plugins and GUIs](Documentation/EditorIntegration.md)
  - [Frequently Asked Questions](Documentation/faq.md)
- [Contributing](CONTRIBUTING.md)
  - [Internal Documentation](Documentation/internal)
  - [API documentation](Documentation/api)
  - [How to write a Delve client](Documentation/api/ClientHowto.md)

Delve is a debugger for the Go programming language. The goal of the project is to provide a simple, full featured debugging tool for Go. Delve should be easy to invoke and easy to use. Chances are if you're using a debugger, things aren't going your way. With that in mind, Delve should stay out of your way as much as possible.
