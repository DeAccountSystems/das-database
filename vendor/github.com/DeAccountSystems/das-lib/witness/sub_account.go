package witness

import (
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/molecule"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
	"github.com/nervosnetwork/ckb-sdk-go/types"
)

type SubAccountBuilder struct {
	Signature          []byte
	PrevRoot           []byte
	CurrentRoot        []byte
	Proof              []byte
	Version            uint32
	SubAccount         *SubAccount
	EditKey            []byte
	EditValue          []byte
	MoleculeSubAccount *molecule.SubAccount
	Account            string
}

type SubAccountParam struct {
	Action      string
	Signature   []byte
	PrevRoot    []byte
	CurrentRoot []byte
	Proof       []byte
	Version     uint32
	SubAccount  *SubAccount
	EditKey     []byte
	EditValue   []byte
}

type SubAccount struct {
	Lock                 *types.Script
	AccountId            string
	AccountCharSet       []*AccountCharSet
	Suffix               string
	RegisteredAt         uint64
	ExpiredAt            uint64
	Status               uint8
	Records              []*SubAccountRecord
	Nonce                uint64
	EnableSubAccount     uint8
	RenewSubAccountPrice uint64
}

func SubAccountDataBuilderFromTx(tx *types.Transaction) (*SubAccountBuilder, error) {
	respMap, err := SubAccountDataBuilderMapFromTx(tx)
	if err != nil {
		return nil, err
	}
	for k, _ := range respMap {
		return respMap[k], nil
	}
	return nil, fmt.Errorf("not exist sub account")
}

func SubAccountIdDataBuilderFromTx(tx *types.Transaction) (map[string]*SubAccountBuilder, error) {
	respMap, err := SubAccountDataBuilderMapFromTx(tx)
	if err != nil {
		return nil, err
	}

	retMap := make(map[string]*SubAccountBuilder)
	for k, v := range respMap {
		k1 := v.SubAccount.AccountId
		retMap[k1] = respMap[k]
	}
	return retMap, nil
}

func SubAccountDataBuilderMapFromTx(tx *types.Transaction) (map[string]*SubAccountBuilder, error) {
	var respMap = make(map[string]*SubAccountBuilder)

	err := GetWitnessDataFromTx(tx, func(actionDataType common.ActionDataType, dataBys []byte) (bool, error) {
		switch actionDataType {
		case common.ActionDataTypeSubAccount:
			var resp SubAccountBuilder
			index, length := 0, 4

			signatureLen, _ := molecule.Bytes2GoU32(dataBys[index:length])
			resp.Signature = dataBys[length:signatureLen]
			index = length + int(signatureLen)

			prevRootLen, _ := molecule.Bytes2GoU32(dataBys[index:length])
			resp.PrevRoot = dataBys[index+length : prevRootLen]
			index = length + int(prevRootLen)

			currentRootLen, _ := molecule.Bytes2GoU32(dataBys[index:length])
			resp.CurrentRoot = dataBys[index+length : currentRootLen]
			index = length + int(currentRootLen)

			proofLen, _ := molecule.Bytes2GoU32(dataBys[index:length])
			resp.Proof = dataBys[index+length : proofLen]
			index = length + int(proofLen)

			versionLen, err := molecule.Bytes2GoU32(dataBys[index:length])
			if err != nil {
				return false, fmt.Errorf("get version len err: %s", err.Error())
			}
			resp.Version, err = molecule.Bytes2GoU32(dataBys[index+length : versionLen])
			if err != nil {
				return false, fmt.Errorf("get version err: %s", err.Error())
			}
			index = length + int(versionLen)

			subAccountLen, _ := molecule.Bytes2GoU32(dataBys[index:length])
			subAccountBys := dataBys[index+length : subAccountLen]
			index = length + int(subAccountLen)

			keyLen, _ := molecule.Bytes2GoU32(dataBys[index:length])
			resp.EditKey = dataBys[index+length : keyLen]
			index = length + int(keyLen)

			valueLen, _ := molecule.Bytes2GoU32(dataBys[index:length])
			resp.EditValue = dataBys[index+length : valueLen]

			switch resp.Version {
			case common.GoDataEntityVersion1:
				subAccount, err := molecule.SubAccountFromSlice(subAccountBys, false)
				if err != nil {
					return false, fmt.Errorf("SubAccountDataFromSlice err: %s", err.Error())
				}
				resp.SubAccount.Lock = molecule.MoleculeScript2CkbScript(subAccount.Lock())
				resp.SubAccount.AccountId = common.Bytes2Hex(subAccount.Id().RawData())
				resp.SubAccount.AccountCharSet = ConvertToAccountCharSets(subAccount.Account())
				resp.SubAccount.Suffix = string(subAccount.Suffix().RawData())
				resp.SubAccount.RegisteredAt, _ = molecule.Bytes2GoU64(subAccount.RegisteredAt().RawData())
				resp.SubAccount.ExpiredAt, _ = molecule.Bytes2GoU64(subAccount.ExpiredAt().RawData())
				resp.SubAccount.Status, _ = molecule.Bytes2GoU8(subAccount.Status().RawData())
				resp.SubAccount.Records = ConvertToSubAccountRecords(subAccount.Records())
				resp.SubAccount.Nonce, _ = molecule.Bytes2GoU64(subAccount.Nonce().RawData())
				resp.SubAccount.EnableSubAccount, _ = molecule.Bytes2GoU8(subAccount.EnableSubAccount().RawData())
				resp.SubAccount.RenewSubAccountPrice, _ = molecule.Bytes2GoU64(subAccount.RenewSubAccountPrice().RawData())
				resp.MoleculeSubAccount = subAccount
				resp.Account = common.AccountCharsToAccount(subAccount.Account())
				respMap[resp.Account] = &resp
			default:
				return false, fmt.Errorf("sub account version: %d", resp.Version)
			}
		}
		return true, nil
	})

	if err != nil {
		return nil, fmt.Errorf("GetWitnessDataFromTx err: %s", err.Error())
	}
	if len(respMap) == 0 {
		return nil, fmt.Errorf("not exist sub account")
	}
	return respMap, nil
}

func (s *SubAccountBuilder) ConvertToSubAccount() *SubAccount {
	var subAccount SubAccount
	switch string(s.EditKey) {
	case "lock":
		lock := s.ConvertEditValueToLock()
		subAccount.Lock = molecule.MoleculeScript2CkbScript(lock)
	case "expired_at":
		expiredAt := s.ConvertEditValueToExpiredAt()
		subAccount.ExpiredAt, _ = molecule.Bytes2GoU64(expiredAt.RawData())
	case "status":
		status := s.ConvertEditValueToStatus()
		subAccount.Status, _ = molecule.Bytes2GoU8(status.RawData())
	case "records":
		records := s.ConvertEditValueToRecords()
		subAccount.Records = ConvertToSubAccountRecords(records)
	case "enable_sub_account":
		enableSubAccount := s.ConvertEditValueToEnableSubAccount()
		subAccount.EnableSubAccount, _ = molecule.Bytes2GoU8(enableSubAccount.RawData())
	case "renew_sub_account_price":
		renewSubAccountPrice := s.ConvertEditValueToRenewSubAccountPrice()
		subAccount.RenewSubAccountPrice, _ = molecule.Bytes2GoU64(renewSubAccountPrice.RawData())
	}
	return &subAccount
}

func (s *SubAccountBuilder) ConvertEditValueToLock() *molecule.Script {
	lock, _ := molecule.ScriptFromSlice(s.EditValue, false)
	return lock
}

func (s *SubAccountBuilder) ConvertEditValueToExpiredAt() *molecule.Uint64 {
	expiredAt, _ := molecule.Uint64FromSlice(s.EditValue, false)
	return expiredAt
}

func (s *SubAccountBuilder) ConvertEditValueToStatus() *molecule.Uint8 {
	status, _ := molecule.Uint8FromSlice(s.EditValue, false)
	return status
}

func (s *SubAccountBuilder) ConvertEditValueToRecords() *molecule.Records {
	records, _ := molecule.RecordsFromSlice(s.EditValue, false)
	return records
}

func (s *SubAccountBuilder) ConvertEditValueToEnableSubAccount() *molecule.Uint8 {
	enableSubAccount, _ := molecule.Uint8FromSlice(s.EditValue, false)
	return enableSubAccount
}

func (s *SubAccountBuilder) ConvertEditValueToRenewSubAccountPrice() *molecule.Uint64 {
	renewSubAccountPrice, _ := molecule.Uint64FromSlice(s.EditValue, false)
	return renewSubAccountPrice
}

type SubAccountRecord struct {
	Key   string
	Type  string
	Label string
	Value string
	TTL   uint32
}

func ConvertToSubAccountRecords(records *molecule.Records) []*SubAccountRecord {
	var subAccountRecords []*SubAccountRecord
	for index, lenRecords := uint(0), records.Len(); index < lenRecords; index++ {
		record := records.Get(index)
		ttl, _ := molecule.Bytes2GoU32(record.RecordTtl().RawData())
		subAccountRecords = append(subAccountRecords, &SubAccountRecord{
			Key:   string(record.RecordKey().RawData()),
			Type:  string(record.RecordType().RawData()),
			Label: string(record.RecordLabel().RawData()),
			Value: string(record.RecordValue().RawData()),
			TTL:   ttl,
		})
	}
	return subAccountRecords
}

func ConvertToRecordsHash(records *molecule.Records) []byte {
	bys, _ := blake2b.Blake256(records.AsSlice())
	return bys
}

type AccountCharType uint32

const (
	AccountCharTypeEmoji  AccountCharType = 0
	AccountCharTypeNumber AccountCharType = 1
	AccountCharTypeEn     AccountCharType = 2
)

type AccountCharSet struct {
	CharSetName AccountCharType `json:"char_set_name"`
	Char        string          `json:"char"`
}

func ConvertToAccountCharSets(accountChars *molecule.AccountChars) []*AccountCharSet {
	index := uint(0)
	var accountCharSets []*AccountCharSet
	for ; index < accountChars.ItemCount(); index++ {
		char := accountChars.Get(index)
		charSetName, _ := molecule.Bytes2GoU32(char.CharSetName().RawData())
		accountCharSets = append(accountCharSets, &AccountCharSet{
			CharSetName: AccountCharType(charSetName),
			Char:        string(char.Bytes().RawData()),
		})
	}
	return accountCharSets
}

/****************************************** Parting Line ******************************************/

func ConvertToAccountChars(accountCharSet []*AccountCharSet) *molecule.AccountChars {
	accountCharsBuilder := molecule.NewAccountCharsBuilder()
	for _, item := range accountCharSet {
		if item.Char == "." {
			break
		}
		accountChar := molecule.NewAccountCharBuilder().
			CharSetName(molecule.GoU32ToMoleculeU32(uint32(item.CharSetName))).
			Bytes(molecule.GoBytes2MoleculeBytes([]byte(item.Char))).Build()
		accountCharsBuilder.Push(accountChar)
	}
	accountChars := accountCharsBuilder.Build()
	return &accountChars
}

func ConvertToRecords(subAccountRecords []*SubAccountRecord) *molecule.Records {
	recordsBuilder := molecule.NewRecordsBuilder()
	for _, v := range subAccountRecords {
		record := molecule.RecordDefault()
		recordBuilder := record.AsBuilder()
		recordBuilder.RecordKey(molecule.GoString2MoleculeBytes(v.Key)).
			RecordType(molecule.GoString2MoleculeBytes(v.Type)).
			RecordLabel(molecule.GoString2MoleculeBytes(v.Label)).
			RecordValue(molecule.GoString2MoleculeBytes(v.Value)).
			RecordTtl(molecule.GoU32ToMoleculeU32(v.TTL))
		recordsBuilder.Push(recordBuilder.Build())
	}
	records := recordsBuilder.Build()
	return &records
}

func (s *SubAccountBuilder) ConvertToMoleculeSubAccount(p *SubAccountParam) *molecule.SubAccount {
	lock := molecule.CkbScript2MoleculeScript(p.SubAccount.Lock)
	accountChars := ConvertToAccountChars(p.SubAccount.AccountCharSet)
	accountId, _ := molecule.AccountIdFromSlice(common.Hex2Bytes(p.SubAccount.AccountId), false)
	suffix := molecule.GoBytes2MoleculeBytes([]byte(p.SubAccount.Suffix))
	registeredAt := molecule.GoU64ToMoleculeU64(p.SubAccount.RegisteredAt)
	expiredAt := molecule.GoU64ToMoleculeU64(p.SubAccount.ExpiredAt)
	status := molecule.GoU8ToMoleculeU8(p.SubAccount.Status)
	records := ConvertToRecords(p.SubAccount.Records)
	nonce := molecule.GoU64ToMoleculeU64(p.SubAccount.Nonce)
	enableSubAccount := molecule.GoU8ToMoleculeU8(p.SubAccount.EnableSubAccount)
	renewSubAccountPrice := molecule.GoU64ToMoleculeU64(p.SubAccount.RenewSubAccountPrice)

	subAccount := molecule.NewSubAccountBuilder().
		Lock(lock).
		Id(*accountId).
		Account(*accountChars).
		Suffix(suffix).
		RegisteredAt(registeredAt).
		ExpiredAt(expiredAt).
		Status(status).
		Records(*records).
		Nonce(nonce).
		EnableSubAccount(enableSubAccount).
		RenewSubAccountPrice(renewSubAccountPrice).
		Build()
	return &subAccount
}

func (s *SubAccountBuilder) genOldSubAccountBytes(p *SubAccountParam, subAccount *molecule.SubAccount) (bys []byte) {
	switch s.Version {
	case common.GoDataEntityVersion1:
		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.Signature)))...)
		bys = append(bys, p.Signature...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.PrevRoot)))...)
		bys = append(bys, p.PrevRoot...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.CurrentRoot)))...)
		bys = append(bys, p.CurrentRoot...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.Proof)))...)
		bys = append(bys, p.Proof...)

		versionBys := molecule.GoU32ToMoleculeU32(s.Version)
		bys = append(bys, molecule.GoU32ToBytes(uint32(len(versionBys.RawData())))...)
		bys = append(bys, versionBys.RawData()...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(subAccount.AsSlice())))...)
		bys = append(bys, subAccount.AsSlice()...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.EditKey)))...)
		bys = append(bys, p.EditKey...)
	}
	return bys
}

func (s *SubAccountBuilder) genNewSubAccountBytes(p *SubAccountParam) (bys []byte) {
	switch p.Version {
	case common.GoDataEntityVersion1:
		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.Signature)))...)
		bys = append(bys, p.Signature...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.PrevRoot)))...)
		bys = append(bys, p.PrevRoot...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.CurrentRoot)))...)
		bys = append(bys, p.CurrentRoot...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.Proof)))...)
		bys = append(bys, p.Proof...)

		versionBys := molecule.GoU32ToMoleculeU32(p.Version)
		bys = append(bys, molecule.GoU32ToBytes(uint32(len(versionBys.RawData())))...)
		bys = append(bys, versionBys.RawData()...)

		moleculeSubAccount := s.ConvertToMoleculeSubAccount(p)
		bys = append(bys, molecule.GoU32ToBytes(uint32(len(moleculeSubAccount.AsSlice())))...)
		bys = append(bys, moleculeSubAccount.AsSlice()...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.EditKey)))...)
		bys = append(bys, p.EditKey...)

		bys = append(bys, molecule.GoU32ToBytes(uint32(len(p.EditValue)))...)
		bys = append(bys, p.EditValue...)
	}
	return bys
}

func (s *SubAccountBuilder) genMoleculeSubAccount() *molecule.SubAccount {
	subAccountBuilder := s.MoleculeSubAccount.AsBuilder()

	// nonce increment on each transaction
	nonceUint64, _ := molecule.Bytes2GoU64(s.MoleculeSubAccount.Nonce().RawData())
	nonceUint64++
	nonce := molecule.GoU64ToMoleculeU64(nonceUint64)

	switch string(s.EditKey) {
	case "lock":
		subAccount := subAccountBuilder.Lock(*s.ConvertEditValueToLock()).Nonce(nonce).Build()
		return &subAccount
	case "expired_at":
		subAccount := subAccountBuilder.ExpiredAt(*s.ConvertEditValueToExpiredAt()).Nonce(nonce).Build()
		return &subAccount
	case "status":
		subAccount := subAccountBuilder.Status(*s.ConvertEditValueToStatus()).Nonce(nonce).Build()
		return &subAccount
	case "records":
		subAccount := subAccountBuilder.Records(*s.ConvertEditValueToRecords()).Nonce(nonce).Build()
		return &subAccount
	case "enable_sub_account":
		subAccount := subAccountBuilder.EnableSubAccount(*s.ConvertEditValueToEnableSubAccount()).Nonce(nonce).Build()
		return &subAccount
	case "renew_sub_account_price":
		subAccount := subAccountBuilder.RenewSubAccountPrice(*s.ConvertEditValueToRenewSubAccountPrice()).Nonce(nonce).Build()
		return &subAccount
	}

	subAccount := subAccountBuilder.Build()
	return &subAccount
}

func (s *SubAccountBuilder) GenWitness(p *SubAccountParam) ([]byte, error) {
	switch p.Action {
	case common.DasActionCreateSubAccount:
		witness := GenDasSubAccountWitness(common.ActionDataTypeSubAccount, s.genNewSubAccountBytes(p))

		return witness, nil
	case common.DasActionEditSubAccount:
		subAccount := s.genMoleculeSubAccount()
		switch string(p.EditKey) {
		case "lock":
			witness := GenDasSubAccountWitness(common.ActionDataTypeSubAccount, s.genOldSubAccountBytes(p, subAccount))
			lock := molecule.CkbScript2MoleculeScript(p.SubAccount.Lock)
			witness = append(witness, molecule.GoU32ToBytes(uint32(len(lock.AsSlice())))...)
			return append(witness, lock.AsSlice()...), nil
		case "status":
			witness := GenDasSubAccountWitness(common.ActionDataTypeSubAccount, s.genOldSubAccountBytes(p, subAccount))
			status := molecule.GoU8ToMoleculeU8(p.SubAccount.Status)
			witness = append(witness, molecule.GoU32ToBytes(uint32(len(status.AsSlice())))...)
			return append(witness, status.RawData()...), nil
		case "records":
			witness := GenDasSubAccountWitness(common.ActionDataTypeSubAccount, s.genOldSubAccountBytes(p, subAccount))
			records := ConvertToRecords(p.SubAccount.Records)
			witness = append(witness, molecule.GoU32ToBytes(uint32(len(records.AsSlice())))...)
			return append(witness, records.AsSlice()...), nil
		case "enable_sub_account":
			witness := GenDasSubAccountWitness(common.ActionDataTypeSubAccount, s.genOldSubAccountBytes(p, subAccount))
			enableSubAccount := molecule.GoU8ToMoleculeU8(p.SubAccount.EnableSubAccount)
			witness = append(witness, molecule.GoU32ToBytes(uint32(len(enableSubAccount.AsSlice())))...)
			return append(witness, enableSubAccount.RawData()...), nil
		case "renew_sub_account_price":
			witness := GenDasSubAccountWitness(common.ActionDataTypeSubAccount, s.genOldSubAccountBytes(p, subAccount))
			renewSubAccountPrice := molecule.GoU64ToMoleculeU64(p.SubAccount.RenewSubAccountPrice)
			witness = append(witness, molecule.GoU32ToBytes(uint32(len(renewSubAccountPrice.AsSlice())))...)
			return append(witness, renewSubAccountPrice.RawData()...), nil
		default:
			return nil, fmt.Errorf("not support edit key [%s]", string(s.EditKey))
		}
	case common.DasActionRenewSubAccount:
		subAccount := s.genMoleculeSubAccount()
		witness := GenDasSubAccountWitness(common.ActionDataTypeSubAccount, s.genOldSubAccountBytes(p, subAccount))
		expiredAt := molecule.GoU64ToMoleculeU64(p.SubAccount.ExpiredAt)
		witness = append(witness, molecule.GoU32ToBytes(uint32(len(expiredAt.RawData())))...)
		return append(witness, expiredAt.RawData()...), nil
	case common.DasActionRecycleSubAccount:
		witness := GenDasSubAccountWitness(common.ActionDataTypeSubAccount, s.genOldSubAccountBytes(p, s.MoleculeSubAccount))
		witness = append(witness, molecule.GoU32ToBytes(uint32(0))...)
		return append(witness, byte(0)), nil
	}
	return nil, fmt.Errorf("not exist action [%s]", p.Action)
}