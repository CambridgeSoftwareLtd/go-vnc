# VNC Library for Go
go-vnc is a VNC client library for Go.

This library implements [RFC 6143][RFC6143] -- The Remote Framebuffer Protocol
-- the protocol used by VNC.

N.B. This is forked from [kward's work](https://github.com/kward/go-vnc) and it an attempt to extend encoding coverage. It is not feature complete or stable at present!

## Project links
* Documentation: [![GoDoc][GoDocStatus]][GoDoc]

## Setup
1. Download software and supporting packages.

    ```
    $ go get github.com/CambridgeSoftwareLtd/go-vnc
    $ go get golang.org/x/net
    ```

## Usage
Sample code usage is available in the GoDoc.

- Connect and listen to server messages: <https://godoc.org/github.com/CambridgeSoftwareLtd/go-vnc#example-Connect>

The source code is laid out such that the files match the document sections:

- [7.1] handshake.go
- [7.2] security.go
- [7.3] initialization.go
- [7.4] pixel_format.go
- [7.5] client.go
- [7.6] server.go
- [7.7] encodings.go

There are two additional files that provide everything else:

- vncclient.go -- code for instantiating a VNC client
- common.go -- common stuff not related to the RFB protocol


<!--- Links -->
[RFC6143]: http://tools.ietf.org/html/rfc6143

[GoDoc]: https://godoc.org/github.com/CambridgeSoftwareLtd/go-vnc
[GoDocStatus]: https://godoc.org/github.com/CambridgeSoftwareLtd/go-vnc?status.svg
