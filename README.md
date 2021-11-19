# Big-tts

![Go](https://github.com/airenas/big-tts/workflows/Go/badge.svg) [![Coverage Status](https://coveralls.io/repos/github/airenas/big-tts/badge.svg?branch=main)](https://coveralls.io/github/airenas/big-tts?branch=main) [![Go Report Card](https://goreportcard.com/badge/github.com/airenas/big-tts)](https://goreportcard.com/report/github.com/airenas/big-tts) ![CodeQL](https://github.com/airenas/big-tts/workflows/CodeQL/badge.svg)

TTS service for large texts.

## Upload text file

```bash
curl -X POST http://localhost:8181/upload -H 'Content-Type: multipart/form-data' -F file=@1.txt
```
