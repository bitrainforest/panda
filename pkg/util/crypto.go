package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func RSAGenKey(bits int) error {
	// private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	privateStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block1 := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateStream,
	}
	fPrivate, err := os.Create("private.pem")
	if err != nil {
		return err
	}
	defer fPrivate.Close()
	err = pem.Encode(fPrivate, &block1)
	if err != nil {
		return err
	}

	// public key
	publicKey := privateKey.PublicKey
	// 使用x509.MarshalPKCS1PublicKey无法解析
	publicStream, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return err
	}
	block2 := pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicStream,
	}
	fPublic, err := os.Create("public.pem")
	if err != nil {
		return err
	}
	defer fPublic.Close()
	pem.Encode(fPublic, &block2)
	return nil
}

func EncryptRSA(src []byte, keyPath string) ([]byte, error) {
	f, err := os.Open(keyPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}
	b := make([]byte, fileInfo.Size())
	f.Read(b)
	block, _ := pem.Decode(b)

	keyInit, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pubKey := keyInit.(*rsa.PublicKey)
	res, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, src)
	return res, err
}

func DecryptRSA(src []byte, path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}
	b := make([]byte, fileInfo.Size())
	f.Read(b)
	block, _ := pem.Decode(b)                                 //解码
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes) //还原数据
	if err != nil {
		return nil, fmt.Errorf("parse private key error: %s", err)
	}
	res, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, src)
	return res, err
}
