// Copyright © 2016 Abcum Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"math/big"
	"net"
	"time"

	"io/ioutil"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"

	"github.com/spf13/cobra"
)

type certClusterOptions struct {
	CA struct {
		Crt string
		Key string
	}
	Out struct {
		Bit int
		Com string
		Org string
		Crt string
		Key string
	}
}

var certClusterOpt *certClusterOptions

var certClusterCmd = &cobra.Command{
	Use:     "cluster",
	Short:   "Create a new cluster certificate and key.",
	Example: "  surreal cert cluster --ca-crt crt/ca.crt --ca-key crt/ca.key --out-crt crt/cluster.crt --out-key crt/cluster.key",
	PreRunE: func(cmd *cobra.Command, args []string) error {

		if len(certClusterOpt.CA.Crt) == 0 {
			return fmt.Errorf("Please provide a CA certificate file path.")
		}

		if len(certClusterOpt.CA.Key) == 0 {
			return fmt.Errorf("Please provide a CA private key file path.")
		}

		if len(certClusterOpt.Out.Org) == 0 {
			return fmt.Errorf("Please provide an organisation name.")
		}

		if len(certClusterOpt.Out.Crt) == 0 {
			return fmt.Errorf("Please provide a certificate file path.")
		}

		if len(certClusterOpt.Out.Key) == 0 {
			return fmt.Errorf("Please provide a private key file path.")
		}

		return nil

	},
	RunE: func(cmd *cobra.Command, args []string) error {

		var enc []byte

		var dns []string
		var ips []net.IP

		for _, v := range args {
			chk := net.ParseIP(v)
			switch {
			case chk.To4() != nil:
				ips = append(ips, chk.To4())
			case chk.To16() != nil:
				ips = append(ips, chk.To16())
			default:
				dns = append(dns, v)
			}
		}

		caCrtFile, err := ioutil.ReadFile(certClusterOpt.CA.Crt)
		if err != nil {
			return fmt.Errorf("Could not read file: %#v", certClusterOpt.CA.Crt)
		}

		caCrtData, _ := pem.Decode(caCrtFile)

		caCrt, err := x509.ParseCertificate(caCrtData.Bytes)
		if err != nil {
			return fmt.Errorf("Could not parse CA certificate: %#v", err)
		}

		caKeyFile, err := ioutil.ReadFile(certClusterOpt.CA.Key)
		if err != nil {
			return fmt.Errorf("Could not read file: %#v", certClusterOpt.CA.Crt)
		}

		caKeyData, _ := pem.Decode(caKeyFile)

		caKey, err := x509.ParsePKCS1PrivateKey(caKeyData.Bytes)
		if err != nil {
			return fmt.Errorf("Could not parse CA private key: %#v", err)
		}

		csr := &x509.Certificate{
			Subject: pkix.Name{
				CommonName:   certClusterOpt.Out.Com,
				Organization: []string{certClusterOpt.Out.Org},
			},
			BasicConstraintsValid: true,
			SignatureAlgorithm:    x509.SHA512WithRSA,
			PublicKeyAlgorithm:    x509.ECDSA,
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(10, 0, 0),
			SerialNumber:          big.NewInt(time.Now().UnixNano()),
			KeyUsage: x509.KeyUsageCertSign |
				x509.KeyUsageDigitalSignature |
				x509.KeyUsageKeyAgreement |
				x509.KeyUsageKeyEncipherment |
				x509.KeyUsageDataEncipherment |
				x509.KeyUsageContentCommitment,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			DNSNames:    dns,
			IPAddresses: ips,
		}

		key, err := rsa.GenerateKey(rand.Reader, certClusterOpt.Out.Bit)
		if err != nil {
			return fmt.Errorf("Certificate generation failed: %#v", err)
		}

		prv := x509.MarshalPKCS1PrivateKey(key)

		pub, err := x509.CreateCertificate(rand.Reader, csr, caCrt, &key.PublicKey, caKey)
		if err != nil {
			return fmt.Errorf("Certificate generation failed: %#v", err)
		}

		enc = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: pub,
		})

		log.Printf("Saving server certificate file into %v...", certClusterOpt.Out.Crt)
		if err := ioutil.WriteFile(certClusterOpt.Out.Crt, enc, 0644); err != nil {
			return fmt.Errorf("Unable to write certificate file to %v: %#v", certClusterOpt.Out.Crt, err)
		}

		enc = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: prv,
		})

		log.Printf("Saving server private key file into %v...", certClusterOpt.Out.Key)
		if err := ioutil.WriteFile(certClusterOpt.Out.Key, enc, 0644); err != nil {
			return fmt.Errorf("Unable to write private key file to %v: %#v", certClusterOpt.Out.Key, err)
		}

		return nil

	},
}

func init() {

	certClusterOpt = &certClusterOptions{}

	certClusterCmd.PersistentFlags().StringVar(&certClusterOpt.CA.Crt, "ca-crt", "ca.crt", "The path to the CA certificate file.")
	certClusterCmd.PersistentFlags().StringVar(&certClusterOpt.CA.Key, "ca-key", "ca.key", "The path to the CA private key file.")

	certClusterCmd.PersistentFlags().IntVar(&certClusterOpt.Out.Bit, "key-size", 4096, "The desired number of bits for the key.")
	certClusterCmd.PersistentFlags().StringVar(&certClusterOpt.Out.Com, "out-com", "", "The common name for the server certificate.")
	certClusterCmd.PersistentFlags().StringVar(&certClusterOpt.Out.Org, "out-org", "", "The origanisation name for the server certificate.")
	certClusterCmd.PersistentFlags().StringVar(&certClusterOpt.Out.Crt, "out-crt", "cluster.crt", "The path destination for the server certificate file.")
	certClusterCmd.PersistentFlags().StringVar(&certClusterOpt.Out.Key, "out-key", "cluster.key", "The path destination for the server private key file.")

}
