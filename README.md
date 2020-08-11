# TRED - Tamper Resistant Encrypted Data
[![Build Status](https://github.com/bryk-io/tred-cli/workflows/ci/badge.svg?branch=master)](https://github.com/bryk-io/tred-cli/actions)
[![Version](https://img.shields.io/github/tag/bryk-io/tred-cli.svg)](https://github.com/bryk-io/tred-cli/releases)
[![Software License](https://img.shields.io/badge/license-BSD3-red.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/bryk-io/tred-cli?style=flat)](https://goreportcard.com/report/github.com/bryk-io/tred-cli)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0-ff69b4.svg)](.github/CODE_OF_CONDUCT.md)

Data protection policies must include __in transit__ and __at rest__ considerations, while
very good open standards exist for secure data transmission the same is not true for local
data persistence. To tackle this need we introduce the `TRED` protocol, a simple, extensible
and performant mechanism to securely manage sensitive data at rest.

Some of its characteristics include:
- Support for modern and robust ciphers [Chacha20](https://en.wikipedia.org/wiki/Salsa20#ChaCha_variant) and [AES256](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard).
- Good performance and little overhead on per-data-packet.
- Prevent manipulation attempts on generated ciphertext.
- Prevent reordering of data packets.
- Prevent leaking information when attempting to process manipulated data packets.
- Prevent overflows when processing large data streams.

You can directly download the binary from the
[published releases](https://github.com/bryk-io/tred-cli/releases).
