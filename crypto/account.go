// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package crypto

import (
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/olm"
	"maunium.net/go/mautrix/id"
)

type OlmAccount struct {
	Internal   olm.Account
	signingKey  id.Curve25519
	identityKey id.Ed25519
	Shared     bool
}

func NewOlmAccount() *OlmAccount {
	return &OlmAccount{
		Internal: *olm.NewAccount(),
	}
}

func (account *OlmAccount) Keys() (id.Ed25519, id.Curve25519) {
	if len(account.signingKey) == 0 || len(account.identityKey) == 0 {
		account.identityKey, account.signingKey = account.Internal.IdentityKeys()
	}
	return account.identityKey, account.signingKey
}

func (account *OlmAccount) SigningKey() id.Curve25519 {
	if len(account.signingKey) == 0 {
		account.identityKey, account.signingKey = account.Internal.IdentityKeys()
	}
	return account.signingKey
}

func (account *OlmAccount) IdentityKey() id.Ed25519 {
	if len(account.identityKey) == 0 {
		account.identityKey, account.signingKey = account.Internal.IdentityKeys()
	}
	return account.identityKey
}

func (account *OlmAccount) getInitialKeys(userID id.UserID, deviceID id.DeviceID) *mautrix.DeviceKeys {
	deviceKeys := &mautrix.DeviceKeys{
		UserID:     userID,
		DeviceID:   deviceID,
		Algorithms: []id.Algorithm{id.AlgorithmMegolmV1, id.AlgorithmOlmV1},
		Keys: map[id.DeviceKeyID]string{
			id.NewDeviceKeyID(id.KeyAlgorithmCurve25519, deviceID): string(account.SigningKey()),
			id.NewDeviceKeyID(id.KeyAlgorithmEd25519, deviceID):    string(account.IdentityKey()),
		},
	}

	signature, err := account.Internal.SignJSON(deviceKeys)
	if err != nil {
		panic(err)
	}

	deviceKeys.Signatures = mautrix.Signatures{
		userID: {
			id.NewDeviceKeyID(id.KeyAlgorithmEd25519, deviceID): signature,
		},
	}
	return deviceKeys
}

func (account *OlmAccount) getOneTimeKeys(userID id.UserID, deviceID id.DeviceID, currentOTKCount int) map[id.KeyID]mautrix.OneTimeKey {
	newCount := int(account.Internal.MaxNumberOfOneTimeKeys()/2) - currentOTKCount
	if newCount > 0 {
		account.Internal.GenOneTimeKeys(uint(newCount))
	}
	oneTimeKeys := make(map[id.KeyID]mautrix.OneTimeKey)
	// TODO do we need unsigned curve25519 one-time keys at all?
	//      this just signs all of them
	for keyID, key := range account.Internal.OneTimeKeys() {
		key := mautrix.OneTimeKey{Key: key}
		signature, _ := account.Internal.SignJSON(key)
		key.Signatures = mautrix.Signatures{
			userID: {
				id.NewDeviceKeyID(id.KeyAlgorithmEd25519, deviceID): signature,
			},
		}
		key.IsSigned = true
		oneTimeKeys[id.NewKeyID(id.KeyAlgorithmSignedCurve25519, keyID)] = key
	}
	account.Internal.MarkKeysAsPublished()
	return oneTimeKeys
}
