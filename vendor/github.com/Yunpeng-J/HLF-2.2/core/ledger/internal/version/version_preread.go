// +build preread

/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/Yunpeng-J/HLF-2.2/common/ledger/util"
)

// Height represents the height of a transaction in blockchain
type Height struct {
	BlockNum uint64
	TxNum    uint64
	Txid     [32]byte
}

// NewHeight constructs a new instance of Height
func NewHeight(blockNum, txNum uint64) *Height {
	return &Height{BlockNum: blockNum, TxNum: txNum}
}

func NewHeightWithTxid(blockNum, txNum uint64, txid string) *Height {
	temp := &Height{BlockNum: blockNum, TxNum: txNum}
	txidbytes, err := hex.DecodeString(txid)
	if err != nil {
		log.Printf("ethereum: DecodeString %s Error = %v\n", txid, err)
		return temp
	}
	copy(temp.Txid[:], txidbytes)
	return temp
}

func NewHeightWithTxidbytes(blockNum, txNum uint64, txid [32]byte) *Height {
	return &Height{BlockNum: blockNum, TxNum: txNum, Txid: txid}
}

// NewHeightFromBytes constructs a new instance of Height from serialized bytes
func NewHeightFromBytes(b []byte) (*Height, int, error) {
	blockNum, n1, err := util.DecodeOrderPreservingVarUint64(b)
	if err != nil {
		return nil, -1, err
	}
	txNum, n2, err := util.DecodeOrderPreservingVarUint64(b[n1:])
	if err != nil {
		return nil, -1, err
	}
	height := NewHeight(blockNum, txNum)
	n3 := len(b) - n1 - n2
	if n3 == 32 {
		copy(height.Txid[:], b[n1+n2:])
	} else if n3 != 0 {
		return height, n1 + n2, fmt.Errorf("ethereum: Unexpected length of Version/Height")
	}
	return height, n1 + n2, nil
}

func (h *Height) SetTxid(txid string) {
	temp, err := hex.DecodeString(txid)
	if err != nil {
		fmt.Printf("ethereum: SetTxid(%s) Error = %v\n", txid, err)
	}
	copy(h.Txid[:], temp)
}

// ToBytes serializes the Height
func (h *Height) ToBytes() []byte {
	blockNumBytes := util.EncodeOrderPreservingVarUint64(h.BlockNum)
	txNumBytes := util.EncodeOrderPreservingVarUint64(h.TxNum)
	temp := append(blockNumBytes, txNumBytes...)
	return append(temp, h.Txid[:]...)
}

// Compare return a -1, zero, or +1 based on whether this height is
// less than, equals to, or greater than the specified height respectively.
func (h *Height) Compare(h1 *Height) int {
	res := 0
	switch {
	case h.BlockNum != h1.BlockNum:
		res = int(h.BlockNum - h1.BlockNum)
	case h.TxNum != h1.TxNum:
		res = int(h.TxNum - h1.TxNum)
	case h.Txid != h1.Txid:
		res = 1 // just mean txid is not equal.
	default:
		return 0
	}
	if res > 0 {
		return 1
	}
	return -1
}

// String returns string for printing
func (h *Height) String() string {
	return fmt.Sprintf("{BlockNum: %d, TxNum: %d, Txid: %s}", h.BlockNum, h.TxNum, hex.EncodeToString(h.Txid[:]))
}

// AreSame returns true if both the heights are either nil or equal
func AreSame(h1 *Height, h2 *Height) bool {
	if h1 == nil {
		return h2 == nil
	}
	if h2 == nil {
		return false
	}
	return h1.Compare(h2) == 0
}
