package quicshim_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"testing"
	"time"

	quicshim "github.com/Warashi/quic-shim"
	"github.com/lucas-clemente/quic-go"
)

func generateClientTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}

func TestStreamConn(t *testing.T) {
	l, err := quicshim.Listen("localhost:0", generateTLSConfig(), &quic.Config{})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := l.Accept()
			t.Log("accepted")
			if err != nil {
				t.Log(err)
			}
			go io.Copy(conn, conn)
		}
	}()
	d := quicshim.NewDialer(generateClientTLSConfig(), &quic.Config{})
	conn, err := d.Dial("", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	buf := new(bytes.Buffer)
	go io.Copy(buf, conn)

	const msg = "hello, world!"
	fmt.Fprint(conn, msg)
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Millisecond)
	if got := buf.String(); got != msg {
		t.Errorf("got=%s, want=%s", got, msg)
	}
}

func TestStreamConnWithHTTP(t *testing.T) {
	l, err := quicshim.Listen("localhost:0", generateTLSConfig(), &quic.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var called bool
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { called = true })
	go http.Serve(l, nil)
	d := quicshim.NewDialer(generateClientTLSConfig(), &quic.Config{})
	c := &http.Client{
		Transport: &http.Transport{
			DialContext: d.DialContext,
		},
	}
	if _, err := c.Get("http://" + l.Addr().String() + "/"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Errorf("handler not called")
	}
}
