# scep

`scep` is a Golang implementation of the Simple Certificate Enrollment Protocol (SCEP).

This package started its life as part of [micromdm/scep](https://github.com/micromdm/scep). 
The core SCEP protocol was extracted from it and is now being maintained by [smallstep](https://smallstep.com).

## Usage

```console
go get github.com/smallstep/scep
```

The package can be used for both client and server operations.

For detailed usage, see the [Go Reference](https://pkg.go.dev/github.com/smallstep/scep).

Example server:

```go
// read a request body containing SCEP message
body, err := ioutil.ReadAll(r.Body)
if err != nil {
    // handle err
}

// parse the SCEP message
msg, err := scep.ParsePKIMessage(body)
if err != nil {
    // handle err
}

// do something with msg
fmt.Println(msg.MessageType)

// extract encrypted pkiEnvelope
err := msg.DecryptPKIEnvelope(CAcert, CAkey)
if err != nil {
    // handle err
}

// use the CSR from decrypted PKCS request and sign
// MyCSRSigner returns an *x509.Certificate here
crt, err := MyCSRSigner(msg.CSRReqMessage.CSR)
if err != nil {
    // handle err
}

// create a CertRep message from the original
certRep, err := msg.Success(CAcert, CAkey, crt)
if err != nil {
    // handle err
}

// send response back
// w is a http.ResponseWriter
w.Write(certRep.Raw)
```
