package block_parser

import (
	"das_database/dao"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
)

func (b *BlockParser) ActionCreateDeviceKeyList(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version transfer account tx")
		return
	}
	log.Info("ActionTransferAccount:", req.BlockNumber, req.TxHash)

	builder, err := witness.WebAuthnKeyListDataBuilderFromTx(req.Tx, common.DataTypeNew)

	//add cidpk
	var cidPk []dao.TableCidPk
	keyList := witness.ConvertToWebauthnKeyList(builder.WebAuthnKeyListData)
	if len(keyList) == 0 {
		resp.Err = fmt.Errorf("ConvertToWebauthnKeyList err: %s", err.Error())
		return
	}
	cidPk = append(cidPk, dao.TableCidPk{
		Cid:             keyList[0].Cid,
		Pk:              keyList[0].PubKey,
		EnableAuthorize: dao.EnableAuthorizeOn,
	})
	if err := b.dbDao.InsertCidPk(cidPk); err != nil {
		resp.Err = fmt.Errorf("InsertCidPk err: %s", err.Error())
	}
	return
}

//add and delete deviceKey
func (b *BlockParser) ActionUpdateDeviceKeyList(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version transfer account tx")
		return
	}
	log.Info("ActionTransferAccount:", req.BlockNumber, req.TxHash)

	builder, err := witness.WebAuthnKeyListDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("WebAuthnKeyListDataBuilderFromTx err: %s", err.Error())
		return
	}
	keyList := witness.ConvertToWebauthnKeyList(builder.WebAuthnKeyListData)
	var master witness.WebauthnKey
	var authorize []dao.TableAuthorize
	var cidPk []dao.TableCidPk
	for i := 0; i < len(keyList); i++ {
		if i == 0 {
			master.MinAlgId = keyList[0].MinAlgId
			master.SubAlgId = keyList[0].SubAlgId
			master.Cid = keyList[0].Cid
			master.PubKey = keyList[0].PubKey
		}
		authorize = append(authorize, dao.TableAuthorize{
			MasterAlgId:    common.DasAlgorithmId(master.MinAlgId),
			MasterSubAlgId: common.DasAlgorithmId(master.SubAlgId),
			MasterCid:      keyList[i].Cid,
			MasterPk:       keyList[i].PubKey,
			SlaveAlgId:     common.DasAlgorithmId(keyList[i].MinAlgId),
			SlaveSubAlgId:  common.DasAlgorithmId(keyList[i].SubAlgId),
			SlaveCid:       keyList[i].Cid,
			SlavePk:        keyList[i].PubKey,
			Outpoint:       common.OutPoint2String(req.TxHash, 0),
		})
		cidPk = append(cidPk, dao.TableCidPk{
			Cid: keyList[i].Cid,
			Pk:  keyList[i].PubKey,
		})
	}
	if err = b.dbDao.UpdateAuthorizeByMaster(authorize, cidPk); err != nil {
		resp.Err = fmt.Errorf("UpdateAuthorizeByMaster err:%s", err.Error())
		return
	}
	return
}
