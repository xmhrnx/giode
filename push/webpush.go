package push

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
)

// Web Push message encryption (RFC 8291, with RFC 8188's aes128gcm
// content coding): the server encrypts each message to the device's
// P-256 key and auth secret, so the push service relaying it can't read
// it. UnifiedPush distributors deliver the encrypted bytes.

// WebPushKeys are a device's Web Push keys, in the URL-safe base64 Web
// Push servers expect: the public key (p256dh) and the auth secret.
type WebPushKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// webPushKey is a device's private key and auth secret.
type webPushKey struct {
	private *ecdh.PrivateKey
	auth    []byte
}

func newWebPushKey() (webPushKey, error) {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return webPushKey{}, err
	}
	auth := make([]byte, 16)
	rand.Read(auth)
	return webPushKey{private: k, auth: auth}, nil
}

func (k webPushKey) public() WebPushKeys {
	enc := base64.RawURLEncoding.EncodeToString
	return WebPushKeys{P256dh: enc(k.private.PublicKey().Bytes()), Auth: enc(k.auth)}
}

var errNotWebPush = errors.New("push: not a Web Push message")

// decrypt decrypts an aes128gcm Web Push message.
func (k webPushKey) decrypt(body []byte) ([]byte, error) {
	// Header: salt (16), record size (4), key id length (1), key id: the
	// server's public key.
	if len(body) < 21 {
		return nil, errNotWebPush
	}
	salt := body[:16]
	rs := binary.BigEndian.Uint32(body[16:20])
	idlen := int(body[20])
	if idlen != 65 || len(body) < 21+idlen+17 || rs < 18 {
		return nil, errNotWebPush
	}
	serverKey, err := ecdh.P256().NewPublicKey(body[21 : 21+idlen])
	if err != nil {
		return nil, errNotWebPush
	}
	shared, err := k.private.ECDH(serverKey)
	if err != nil {
		return nil, err
	}
	info := append([]byte("WebPush: info\x00"), k.private.PublicKey().Bytes()...)
	info = append(info, serverKey.Bytes()...)
	ikm, err := hkdf.Key(sha256.New, shared, k.auth, string(info), 32)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	var out []byte
	data := body[21+idlen:]
	for seq := uint64(0); len(data) > 0; seq++ {
		n := min(int(rs), len(data))
		rec := data[:n]
		data = data[n:]
		iv := bytes.Clone(nonce)
		for i := 0; i < 8; i++ {
			iv[11-i] ^= byte(seq >> (8 * i))
		}
		plain, err := gcm.Open(nil, iv, rec, nil)
		if err != nil {
			return nil, err
		}
		// Padding: a delimiter (2 for the last record, 1 otherwise),
		// then zeros.
		plain = bytes.TrimRight(plain, "\x00")
		if len(plain) == 0 {
			return nil, errors.New("push: bad padding")
		}
		last := plain[len(plain)-1] == 2
		out = append(out, plain[:len(plain)-1]...)
		if last {
			break
		}
	}
	return out, nil
}

// encryptWebPush encrypts plain to the device's keys, as a server
// does: for tests and for apps relaying to other devices.
func encryptWebPush(keys WebPushKeys, plain []byte) ([]byte, error) {
	dec := base64.RawURLEncoding.DecodeString
	pub, err := dec(keys.P256dh)
	if err != nil {
		return nil, err
	}
	auth, err := dec(keys.Auth)
	if err != nil {
		return nil, err
	}
	ua, err := ecdh.P256().NewPublicKey(pub)
	if err != nil {
		return nil, err
	}
	as, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	shared, err := as.ECDH(ua)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	rand.Read(salt)
	info := append([]byte("WebPush: info\x00"), ua.Bytes()...)
	info = append(info, as.PublicKey().Bytes()...)
	ikm, _ := hkdf.Key(sha256.New, shared, auth, string(info), 32)
	cek, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	rec := gcm.Seal(nil, nonce, append(bytes.Clone(plain), 2), nil)
	var b bytes.Buffer
	b.Write(salt)
	binary.Write(&b, binary.BigEndian, uint32(4096))
	b.WriteByte(65)
	b.Write(as.PublicKey().Bytes())
	b.Write(rec)
	return b.Bytes(), nil
}

func ecdhP256Private(b []byte) (*ecdh.PrivateKey, error) { return ecdh.P256().NewPrivateKey(b) }
