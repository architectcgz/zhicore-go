package publicid

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	MaxLength      = 32
	checksumLength = 4
	alphabet       = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
)

var (
	ErrInvalidFormat   = errors.New("invalid notification public id format")
	ErrInvalidPrefix   = errors.New("invalid notification public id prefix")
	ErrUnknownVersion  = errors.New("unknown notification public id version")
	ErrInvalidChecksum = errors.New("invalid notification public id checksum")
)

type Config struct {
	Prefix        string
	ActiveVersion uint8
	Secrets       map[uint8]string
}

type Codec struct {
	prefix        string
	activeVersion uint8
	secrets       map[uint8]uint64
}

func NewCodec(config Config) (*Codec, error) {
	prefix := strings.TrimSpace(config.Prefix)
	if prefix == "" {
		return nil, fmt.Errorf("notification public id prefix is required")
	}
	if config.ActiveVersion == 0 || config.ActiveVersion > 9 {
		return nil, fmt.Errorf("notification public id active version must be 1-9")
	}
	if strings.TrimSpace(config.Secrets[config.ActiveVersion]) == "" {
		return nil, fmt.Errorf("notification public id active secret is required")
	}

	secrets := make(map[uint8]uint64, len(config.Secrets))
	for version, secret := range config.Secrets {
		if version == 0 || version > 9 {
			return nil, fmt.Errorf("notification public id secret version must be 1-9")
		}
		if strings.TrimSpace(secret) == "" {
			return nil, fmt.Errorf("notification public id secret for version %d is required", version)
		}
		secrets[version] = deriveSecret(version, secret)
	}

	return &Codec{
		prefix:        prefix,
		activeVersion: config.ActiveVersion,
		secrets:       secrets,
	}, nil
}

func (c *Codec) Encode(id uint64) (string, error) {
	if id == 0 {
		return "", fmt.Errorf("%w: id must be positive", ErrInvalidFormat)
	}
	secret, ok := c.secrets[c.activeVersion]
	if !ok {
		return "", ErrUnknownVersion
	}

	code := base58Encode(permute64(id, secret))
	version := versionByte(c.activeVersion)
	raw := c.prefix + string(version) + code
	encoded := raw + checksum(raw, secret)
	if len(encoded) > MaxLength {
		return "", fmt.Errorf("%w: encoded notification public id exceeds max length", ErrInvalidFormat)
	}
	return encoded, nil
}

func (c *Codec) Decode(publicID string) (uint64, error) {
	if publicID == "" || len(publicID) <= len(c.prefix)+1+checksumLength {
		return 0, ErrInvalidFormat
	}
	if !strings.HasPrefix(publicID, c.prefix) {
		return 0, ErrInvalidPrefix
	}

	version, ok := parseVersion(publicID[len(c.prefix)])
	if !ok {
		return 0, ErrInvalidFormat
	}
	secret, ok := c.secrets[version]
	if !ok {
		return 0, ErrUnknownVersion
	}

	bodyWithChecksum := publicID[len(c.prefix)+1:]
	code := bodyWithChecksum[:len(bodyWithChecksum)-checksumLength]
	gotChecksum := bodyWithChecksum[len(bodyWithChecksum)-checksumLength:]
	if !base58Valid(code) || !base58Valid(gotChecksum) {
		return 0, ErrInvalidFormat
	}

	raw := c.prefix + string(versionByte(version)) + code
	if checksum(raw, secret) != gotChecksum {
		return 0, ErrInvalidChecksum
	}
	permuted, ok := base58Decode(code)
	if !ok {
		return 0, ErrInvalidFormat
	}
	id := unpermute64(permuted, secret)
	if id == 0 {
		return 0, ErrInvalidFormat
	}
	return id, nil
}

func deriveSecret(version uint8, secret string) uint64 {
	sum := sha256.Sum256([]byte(fmt.Sprintf("notification-public-id:%d:%s", version, secret)))
	return binary.BigEndian.Uint64(sum[:8])
}

func checksum(raw string, secret uint64) string {
	var secretBytes [8]byte
	binary.BigEndian.PutUint64(secretBytes[:], secret)
	sum := sha256.Sum256(append([]byte(raw), secretBytes[:]...))
	value := uint64(sum[0])<<16 | uint64(sum[1])<<8 | uint64(sum[2])
	out := base58Encode(value)
	for len(out) < checksumLength {
		out = "1" + out
	}
	if len(out) > checksumLength {
		return out[len(out)-checksumLength:]
	}
	return out
}

func permute64(value, secret uint64) uint64 {
	left := uint32(value >> 32)
	right := uint32(value)
	keys := roundKeys(secret)
	for _, key := range keys {
		nextLeft := right
		nextRight := left ^ feistel(right, key)
		left, right = nextLeft, nextRight
	}
	return uint64(left)<<32 | uint64(right)
}

func unpermute64(value, secret uint64) uint64 {
	left := uint32(value >> 32)
	right := uint32(value)
	keys := roundKeys(secret)
	for i := len(keys) - 1; i >= 0; i-- {
		previousRight := left
		previousLeft := right ^ feistel(left, keys[i])
		left, right = previousLeft, previousRight
	}
	return uint64(left)<<32 | uint64(right)
}

func roundKeys(secret uint64) [4]uint32 {
	var keys [4]uint32
	var secretBytes [8]byte
	binary.BigEndian.PutUint64(secretBytes[:], secret)
	sum := sha256.Sum256(append([]byte("notification-public-id-rounds:"), secretBytes[:]...))
	for i := range keys {
		keys[i] = binary.BigEndian.Uint32(sum[i*4 : i*4+4])
	}
	return keys
}

func feistel(value, key uint32) uint32 {
	x := value ^ key
	x ^= x >> 16
	x *= 0x7feb352d
	x ^= x >> 15
	x *= 0x846ca68b
	x ^= x >> 16
	return x
}

func base58Encode(value uint64) string {
	if value == 0 {
		return string(alphabet[0])
	}
	var out []byte
	for value > 0 {
		remainder := value % 58
		out = append(out, alphabet[remainder])
		value /= 58
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

func base58Decode(input string) (uint64, bool) {
	var value uint64
	for _, char := range input {
		index := strings.IndexRune(alphabet, char)
		if index < 0 {
			return 0, false
		}
		value = value*58 + uint64(index)
	}
	return value, true
}

func base58Valid(input string) bool {
	if input == "" {
		return false
	}
	for _, char := range input {
		if !strings.ContainsRune(alphabet, char) {
			return false
		}
	}
	return true
}

func versionByte(version uint8) byte {
	return byte('0' + version)
}

func parseVersion(input byte) (uint8, bool) {
	if input < '1' || input > '9' {
		return 0, false
	}
	return uint8(input - '0'), true
}
