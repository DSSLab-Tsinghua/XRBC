package reedsolomon

import (
	"fmt"

	"github.com/vivint/infectious" //TODO:repace this with a more efficient implementation
)

// RScoder is a reedsolomon coder
type RScoder struct {
	fec *infectious.FEC
}

// NewRScoder returns a RScoder object
func NewRScoder(required, total int) (*RScoder, error) {
	fec, err := infectious.NewFEC(required, total)
	if err != nil {
		return nil, fmt.Errorf("NewFEC failed: %v", err)
	}
	if fec == nil {
		return nil, fmt.Errorf("FEC object is nil")
	}
	return &RScoder{fec: fec}, nil
}

// Encode returns shares of the encoded message
// input: message, n-number of nodes, t-threshold
// output: a list of shares
func (coder *RScoder) Encode(msg []byte) [][]byte {
	// Ensure the message is padded to a multiple of the required size
	paddingLength := coder.fec.Required() - (len(msg) % coder.fec.Required())
	if paddingLength == coder.fec.Required() {
		paddingLength = 0
	}
	paddingMessage := make([]byte, len(msg)+paddingLength)
	copy(paddingMessage, msg)
	if paddingLength > 0 {
		paddingMessage[len(paddingMessage)-1] = byte(paddingLength)
	}

	// Prepare to receive the shares
	shares := make([]infectious.Share, coder.fec.Total())
	output := func(s infectious.Share) {
		shares[s.Number] = s.DeepCopy()
	}

	// Encode the padded message
	err := coder.fec.Encode(paddingMessage, output)
	if err != nil {
		panic(err)
	}

	// Convert shares to [][]byte
	result := make([][]byte, len(shares))
	for i, share := range shares {
		result[i] = share.Data
	}

	return result
}

// func (coder *RScoder) Encode(msg []byte) []infectious.Share {
// 	shares := make([]infectious.Share, coder.fec.Total())
// 	output := func(s infectious.Share) {
// 		shares[s.Number] = s.DeepCopy() // the memory in s gets reused, so we need to make a deep copy
// 	}

// 	paddingLength := coder.fec.Required() - (len(msg) % coder.fec.Required())
// 	paddingMessage := make([]byte, len(msg)+paddingLength)
// 	copy(paddingMessage, msg)
// 	paddingMessage[len(paddingMessage)-1] = byte(paddingLength) //p.F+1 == coder.Required() == paddingLength < 256, so byte is enough
// 	err := coder.fec.Encode(paddingMessage, output)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return shares
// }

// Decode returns the original message of the shares
// input: a list of shares [][]byte
// output: the original message
func (coder *RScoder) Decode(shares [][]byte) ([]byte, error) {
	if coder.fec == nil {
		return nil, fmt.Errorf("RScoder.fec is nil (was NewRScoder called correctly?)")
	}

	// Decode the shares to reconstruct the original padded message
	sharesList := make([]infectious.Share, len(shares))

	if len(shares) < coder.fec.Required() {
		return nil, fmt.Errorf("insufficient shares: got %d, need at least %d", len(shares), coder.fec.Required())
	}

	for i, share := range shares {
		// if share == nil {
		// 	return nil, fmt.Errorf("share %d is nil", i)
		// }
		sharesList[i] = infectious.Share{
			Number: int(i),
			Data:   share,
		}
	}
	result, err := coder.fec.Decode(nil, sharesList)
	if err != nil {
		return nil, err
	}

	// Remove the padding based on the last byte
	paddingLength := int(result[len(result)-1])
	if paddingLength > len(result) {
		return nil, fmt.Errorf("invalid padding length")
	}
	return result[:len(result)-paddingLength], nil
}
