// Copyright (C) 2014 Ryan Hileman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"math/big"
	"time"
	"io/ioutil"

	. "github.com/lipixun/openssl"
	)

func main() {

	TestPKCS12MarshalUnmarshal()
	TestPKCS12MarshalUnmarshalWithCa()

	fmt.Println("OK")
}

func TestPKCS12MarshalUnmarshal() {
	key, err := GenerateRSAKey(2048)
	if err != nil {
		panic(err)
	}
	info := &CertificateInfo{
		Serial:       big.NewInt(int64(1)),
		Issued:       0,
		Expires:      24 * time.Hour,
		Country:      "US",
		Organization: "Test",
		CommonName:   "localhost",
	}
	cert, err := NewCertificate(info, key)
	if err != nil {
		panic(err)
	}
	if err := cert.Sign(key, EVP_SHA256); err != nil {
		panic(err)
	}
	pkcs12 := &PKCS12{
		Name:        "A Friendly Name ^_^",
		Certificate: cert,
		PrivateKey:  key,
	}
	var bytes []byte
	var loadedPKCS12 *PKCS12
	// Test marshal pkcs12 without password
	bytes, err = pkcs12.Marshal("")
	if err != nil {
		panic(err)
	}
	loadedPKCS12, err = UnmarshalPKCS12(bytes, "")
	if err != nil {
		panic(err)
	}
	if ok, reason := isPKCS12Equals(loadedPKCS12, pkcs12); !ok {
		panic(reason)
	}
	if _, err = UnmarshalPKCS12(bytes, "awrongpassword"); err == nil {
		panic("Mac error should be happend")
	}
	// Test marshal pkcs12 with password
	bytes, err = pkcs12.Marshal("apasswordfortest")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile("/tmp/p12", bytes, 0600)

	if _, err = UnmarshalPKCS12(bytes, ""); err == nil {
		panic("Mac error should be happend")
	}
	if _, err = UnmarshalPKCS12(bytes, "awrongpassword"); err == nil {
		panic("Mac error should be happend")
	}
	loadedPKCS12, err = UnmarshalPKCS12(bytes, "apasswordfortest")
	if err != nil {
		panic(err)
	}
	if ok, reason := isPKCS12Equals(loadedPKCS12, pkcs12); !ok {
		panic(reason)
	}
}

func TestPKCS12MarshalUnmarshalWithCa() {
	cakey, err := GenerateRSAKey(2048)
	if err != nil {
		panic(err)
	}
	info := &CertificateInfo{
		Serial:       big.NewInt(int64(1)),
		Issued:       0,
		Expires:      24 * time.Hour,
		Country:      "US",
		Organization: "Test CA",
		CommonName:   "CA",
	}
	ca, err := NewCertificate(info, cakey)
	if err != nil {
		panic(err)
	}
	if err := ca.AddExtensions(map[NID]string{
		NID_basic_constraints:      "critical,CA:TRUE",
		NID_key_usage:              "critical,keyCertSign,cRLSign",
		NID_subject_key_identifier: "hash",
		NID_netscape_cert_type:     "sslCA",
	}); err != nil {
		panic(err)
	}
	if err := ca.Sign(cakey, EVP_SHA256); err != nil {
		panic(err)
	}
	key, err := GenerateRSAKey(2048)
	if err != nil {
		panic(err)
	}
	info = &CertificateInfo{
		Serial:       big.NewInt(int64(2)),
		Issued:       0,
		Expires:      24 * time.Hour,
		Country:      "US",
		Organization: "Test",
		CommonName:   "localhost",
	}
	cert, err := NewCertificate(info, key)
	if err != nil {
		panic(err)
	}
	if err := cert.AddExtensions(map[NID]string{
		NID_basic_constraints: "critical,CA:FALSE",
		NID_key_usage:         "keyEncipherment",
		NID_ext_key_usage:     "serverAuth",
	}); err != nil {
		panic(err)
	}
	if err := cert.SetIssuer(ca); err != nil {
		panic(err)
	}
	if err := cert.Sign(cakey, EVP_SHA256); err != nil {
		panic(err)
	}
	pkcs12 := &PKCS12{
		Name:        "Another Friendly Name ^_^",
		Certificate: cert,
		PrivateKey:  key,
		CaCerts:     []*Certificate{ca},
	}
	var bytes []byte
	var loadedPKCS12 *PKCS12
	bytes, err = pkcs12.Marshal("apassword")
	if err != nil {
		panic(err)
	}
	loadedPKCS12, err = UnmarshalPKCS12(bytes, "apassword")
	if err != nil {
		panic(err)
	}
	if ok, reason := isPKCS12Equals(loadedPKCS12, pkcs12); !ok {
		panic(reason)
	}
}

func isPKCS12Equals(actual, expect *PKCS12) (bool, string) {
	if actual.Name != expect.Name {
		return false, fmt.Sprint("PKCS12 name mismatch. Expect:", expect.Name, "Actual:", actual.Name)
	}
	if len(actual.CaCerts) != len(expect.CaCerts) {
		return false, fmt.Sprint("PKCS12 ca certificas mismatch. Expect length:", len(expect.CaCerts), "Actual length:", len(actual.CaCerts))
	}
	// TODO More validations should be implemented
	// Good
	return true, ""
}
