# Nym Validator

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://github.com/nymtech/nym-validator/blob/master/LICENSE)
<!-- [![Build Status](https://travis-ci.com/jstuczyn/CoconutGo.svg?branch=master)](https://travis-ci.com/jstuczyn/CoconutGo)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/0xacab.org/jstuczyn/CoconutGo)
[![Coverage Status](http://codecov.io/github/jstuczyn/CoconutGo/coverage.svg?branch=master)](http://codecov.io/github/jstuczyn/CoconutGo?branch=master) -->

**DEPRECATED: we are re-writing our validator code in Rust. This codebase is now retired, if you want to set up a validator please build it from the main [Nym platform monorepo](https://github.com/nymtech/nym). We leave this code here in case anybody wants a look at a Coconut implementation in Go.**

This is the Nym validator, associated client code, and dummy service provider code.

It contains a Go implementation of the [Coconut](https://arxiv.org/pdf/1802.07344.pdf) selective disclosure credentials scheme. Coconut supports threshold issuance on multiple public and private attributes, re-randomization and multiple unlinkable selective attribute revelations.

For more information, see the [documentation](https://nymtech.net/docs/).

