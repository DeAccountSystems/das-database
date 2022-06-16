package block_parser

import (
	"das_database/dao"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
	"github.com/DeAccountSystems/das-lib/witness"
	"github.com/scorpiotzh/toolib"
	"strconv"
	"time"
)

func (b *BlockParser) ActionEditRecords(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version edit records tx")
		return
	}
	log.Info("ActionEditRecords:", req.BlockNumber, req.TxHash)

	accBuilder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	var recordsInfos []dao.TableRecordsInfo
	account := accBuilder.Account
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	recordList := accBuilder.RecordList()
	for _, v := range recordList {
		recordsInfos = append(recordsInfos, dao.TableRecordsInfo{
			AccountId: accountId,
			Account:   account,
			Key:       v.Key,
			Type:      v.Type,
			Label:     v.Label,
			Value:     v.Value,
			Ttl:       strconv.FormatUint(uint64(v.TTL), 10),
		})
	}
	accountInfo := dao.TableAccountInfo{
		BlockNumber: req.BlockNumber,
		Outpoint:    common.OutPoint2String(req.TxHash, uint(accBuilder.Index)),
		Account:     account,
		AccountId:   accountId,
	}
	_, mHex, err := b.dasCore.Daf().ArgsToHex(req.Tx.Outputs[accBuilder.Index].Lock.Args)
	if err != nil {
		resp.Err = fmt.Errorf("ArgsToHex err: %s", err.Error())
		return
	}

	transactionInfo := dao.TableTransactionInfo{
		BlockNumber:    req.BlockNumber,
		AccountId:      accountId,
		Account:        account,
		Action:         common.DasActionEditRecords,
		ServiceType:    dao.ServiceTypeRegister,
		ChainType:      mHex.ChainType,
		Address:        mHex.AddressHex,
		Capacity:       0,
		Outpoint:       common.OutPoint2String(req.TxHash, uint(accBuilder.Index)),
		BlockTimestamp: req.BlockTimestamp,
	}

	log.Info("ActionEditRecords:", account, transactionInfo.Address)

	if err := b.dbDao.CreateRecordsInfos(accountInfo, recordsInfos, transactionInfo); err != nil {
		log.Error("CreateRecordsInfos err:", err.Error(), toolib.JsonString(transactionInfo))
		resp.Err = fmt.Errorf("CreateRecordsInfos err: %s", err.Error())
	}

	return
}

func (b *BlockParser) ActionEditManager(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version edit manager tx")
		return
	}
	log.Info("ActionEditManager:", req.BlockNumber, req.TxHash)

	accBuilder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	account := accBuilder.Account
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	ownerHex, managerHex, err := b.dasCore.Daf().ArgsToHex(req.Tx.Outputs[accBuilder.Index].Lock.Args)
	if err != nil {
		resp.Err = fmt.Errorf("ArgsToHex err: %s", err.Error())
		return
	}
	transactionInfo := dao.TableTransactionInfo{
		BlockNumber:    req.BlockNumber,
		AccountId:      accountId,
		Account:        account,
		Action:         common.DasActionEditManager,
		ServiceType:    dao.ServiceTypeRegister,
		ChainType:      ownerHex.ChainType,
		Address:        ownerHex.AddressHex,
		Capacity:       0,
		Outpoint:       common.OutPoint2String(req.TxHash, uint(accBuilder.Index)),
		BlockTimestamp: req.BlockTimestamp,
	}
	accountInfo := dao.TableAccountInfo{
		BlockNumber:        req.BlockNumber,
		Outpoint:           common.OutPoint2String(req.TxHash, uint(accBuilder.Index)),
		Account:            account,
		AccountId:          accountId,
		ManagerChainType:   managerHex.ChainType,
		Manager:            managerHex.AddressHex,
		ManagerAlgorithmId: managerHex.DasAlgorithmId,
	}

	log.Info("ActionEditManager:", account, managerHex.DasAlgorithmId, managerHex.ChainType, managerHex.AddressHex, transactionInfo.Address)

	if err := b.dbDao.EditManager(accountInfo, transactionInfo); err != nil {
		log.Error("EditManager err:", err.Error(), toolib.JsonString(transactionInfo))
		resp.Err = fmt.Errorf("EditManager err: %s", err.Error())
	}

	return
}

func (b *BlockParser) ActionRenewAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version renew account tx")
		return
	}
	log.Info("ActionRenewAccount:", req.BlockNumber, req.TxHash)

	incomeContract, err := core.GetDasContractInfo(common.DasContractNameIncomeCellType)
	if err != nil {
		resp.Err = fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		return
	}

	var inputsOutpoints []string
	var incomeCellInfos []dao.TableIncomeCellInfo
	for _, v := range req.Tx.Inputs {
		inputsOutpoints = append(inputsOutpoints, common.OutPoint2String(v.PreviousOutput.TxHash.Hex(), v.PreviousOutput.Index))
	}
	renewCapacity := uint64(0)
	for i, v := range req.Tx.Outputs {
		if v.Type == nil {
			continue
		}
		if incomeContract.IsSameTypeId(v.Type.CodeHash) {
			renewCapacity = v.Capacity
			incomeCellInfos = append(incomeCellInfos, dao.TableIncomeCellInfo{
				BlockNumber:    req.BlockNumber,
				Action:         common.DasActionRenewAccount,
				Outpoint:       common.OutPoint2String(req.TxHash, uint(i)),
				Capacity:       v.Capacity,
				BlockTimestamp: req.BlockTimestamp,
				Status:         dao.IncomeCellStatusUnMerge,
			})
		}
	}

	builder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(builder.Account))
	accountInfo := dao.TableAccountInfo{
		BlockNumber: req.BlockNumber,
		Outpoint:    common.OutPoint2String(req.TxHash, uint(builder.Index)),
		AccountId:   accountId,
		Account:     builder.Account,
		ExpiredAt:   builder.ExpiredAt,
	}

	ownerHex, _, err := b.dasCore.Daf().ArgsToHex(req.Tx.Outputs[builder.Index].Lock.Args)
	if err != nil {
		resp.Err = fmt.Errorf("ArgsToHex err: %s", err.Error())
		return
	}
	transactionInfo := dao.TableTransactionInfo{
		BlockNumber:    req.BlockNumber,
		AccountId:      accountId,
		Account:        builder.Account,
		Action:         common.DasActionRenewAccount,
		ServiceType:    dao.ServiceTypeRegister,
		ChainType:      ownerHex.ChainType,
		Address:        ownerHex.AddressHex,
		Capacity:       renewCapacity,
		Outpoint:       common.OutPoint2String(req.TxHash, uint(builder.Index)),
		BlockTimestamp: req.BlockTimestamp,
	}

	log.Info("ActionRenewAccount:", builder.Account, builder.ExpiredAt, transactionInfo.Capacity)

	if err := b.dbDao.RenewAccount(inputsOutpoints, incomeCellInfos, accountInfo, transactionInfo); err != nil {
		log.Error("RenewAccount err:", err.Error(), toolib.JsonString(transactionInfo))
		resp.Err = fmt.Errorf("RenewAccount err: %s", err.Error())
	}

	return
}

func (b *BlockParser) ActionTransferAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version transfer account tx")
		return
	}
	log.Info("ActionTransferAccount:", req.BlockNumber, req.TxHash)

	builder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	account := builder.Account
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))

	oHex, mHex, err := b.dasCore.Daf().ArgsToHex(req.Tx.Outputs[builder.Index].Lock.Args)
	if err != nil {
		resp.Err = fmt.Errorf("ArgsToHex err: %s", err.Error())
		return
	}
	oldBuilder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeOld)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	res, err := b.ckbClient.GetTxByHashOnChain(req.Tx.Inputs[oldBuilder.Index].PreviousOutput.TxHash)
	if err != nil {
		resp.Err = fmt.Errorf("GetTxByHashOnChain err: %s", err.Error())
		return
	}

	oldHex, _, err := b.dasCore.Daf().ArgsToHex(res.Transaction.Outputs[req.Tx.Inputs[oldBuilder.Index].PreviousOutput.Index].Lock.Args)
	if err != nil {
		resp.Err = fmt.Errorf("ArgsToHex err: %s", err.Error())
		return
	}
	transactionInfo := dao.TableTransactionInfo{
		BlockNumber:    req.BlockNumber,
		AccountId:      accountId,
		Account:        account,
		Action:         common.DasActionTransferAccount,
		ServiceType:    dao.ServiceTypeRegister,
		ChainType:      oldHex.ChainType,
		Address:        oldHex.AddressHex,
		Capacity:       0,
		Outpoint:       common.OutPoint2String(req.TxHash, uint(builder.Index)),
		BlockTimestamp: req.BlockTimestamp,
	}
	accountInfo := dao.TableAccountInfo{
		BlockNumber:        req.BlockNumber,
		Outpoint:           common.OutPoint2String(req.TxHash, uint(builder.Index)),
		AccountId:          accountId,
		Account:            account,
		OwnerChainType:     oHex.ChainType,
		Owner:              oHex.AddressHex,
		OwnerAlgorithmId:   oHex.DasAlgorithmId,
		ManagerChainType:   mHex.ChainType,
		Manager:            mHex.AddressHex,
		ManagerAlgorithmId: mHex.DasAlgorithmId,
	}
	var recordsInfos []dao.TableRecordsInfo
	recordList := builder.RecordList()
	for _, v := range recordList {
		recordsInfos = append(recordsInfos, dao.TableRecordsInfo{
			AccountId: accountId,
			Account:   account,
			Key:       v.Key,
			Type:      v.Type,
			Label:     v.Label,
			Value:     v.Value,
			Ttl:       strconv.FormatUint(uint64(v.TTL), 10),
		})
	}

	log.Info("ActionTransferAccount:", account, oHex.DasAlgorithmId, oHex.ChainType, oHex.AddressHex, mHex.DasAlgorithmId, mHex.ChainType, mHex.AddressHex, transactionInfo.Address)

	if err := b.dbDao.TransferAccount(accountInfo, transactionInfo, recordsInfos); err != nil {
		log.Error("TransferAccount err:", err.Error(), toolib.JsonString(transactionInfo))
		resp.Err = fmt.Errorf("TransferAccount err: %s", err.Error())
	}

	return
}

func (b *BlockParser) ActionRecycleExpiredAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version account cross chain tx")
		return
	}
	log.Info("ActionRecycleExpiredAccount:", req.BlockNumber, req.TxHash)

	res, err := b.ckbClient.GetTxByHashOnChain(req.Tx.Inputs[1].PreviousOutput.TxHash)
	if err != nil {
		resp.Err = fmt.Errorf("GetTxByHashOnChain err: %s", err.Error())
		return
	}
	builder, err := witness.AccountCellDataBuilderFromTx(res.Transaction, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	builderConfig, err := b.dasCore.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsAccount)
	if err != nil {
		resp.Err = fmt.Errorf("ConfigCellDataBuilderByTypeArgs err: %s", err.Error())
		return
	}
	gracePeriod, err := builderConfig.ExpirationGracePeriod()
	if err != nil {
		resp.Err = fmt.Errorf("ExpirationGracePeriod err: %s", err.Error())
		return
	}

	if builder.ExpiredAt+uint64(gracePeriod) > uint64(time.Now().Unix()) {
		resp.Err = fmt.Errorf("ActionRecycleExpiredAccount: account has not expired yet")
		return
	}
	oHex, _, err := b.dasCore.Daf().ArgsToHex(res.Transaction.Outputs[builder.Index].Lock.Args)
	if err != nil {
		resp.Err = fmt.Errorf("ArgsToHex err: %s", err.Error())
		return
	}
	transactionInfo := dao.TableTransactionInfo{
		BlockNumber:    req.BlockNumber,
		AccountId:      builder.AccountId,
		Account:        builder.Account,
		Action:         common.DasActionRecycleExpiredAccount,
		ServiceType:    dao.ServiceTypeRegister,
		ChainType:      oHex.ChainType,
		Address:        oHex.AddressHex,
		Capacity:       req.Tx.OutputsCapacity() - req.Tx.Outputs[0].Capacity,
		Outpoint:       common.OutPoint2String(req.TxHash, 0),
		BlockTimestamp: req.BlockTimestamp,
	}
	var subAccountIds []string
	if builder.EnableSubAccount == 1 {
		accountInfos, err := b.dbDao.GetAccountInfoByParentAccountId(builder.AccountId)
		if err != nil {
			resp.Err = fmt.Errorf("GetAccountInfoByParentAccountId err: %s", err.Error())
			return
		}
		for _, accountInfo := range accountInfos {
			subAccountIds = append(subAccountIds, accountInfo.AccountId)
		}
	}

	log.Info("ActionRecycleExpiredAccount:", builder.Account, oHex.DasAlgorithmId, oHex.ChainType, oHex.AddressHex, len(subAccountIds))

	if err = b.dbDao.RecycleExpiredAccount(builder.AccountId, subAccountIds, transactionInfo); err != nil {
		resp.Err = fmt.Errorf("RecycleExpiredAccount err: %s", err.Error())
		return
	}

	return
}

func (b *BlockParser) ActionAccountCrossChain(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version account cross chain tx")
		return
	}
	log.Info("ActionAccountCrossChain:", req.BlockNumber, req.TxHash, req.Action)

	accBuilder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	status := dao.AccountStatusOnLock
	if req.Action == common.DasActionUnlockAccountForCrossChain {
		status = dao.AccountStatusNormal
	}

	ownerHex, managerHex, err := b.dasCore.Daf().ArgsToHex(req.Tx.Outputs[0].Lock.Args)
	if err != nil {
		resp.Err = fmt.Errorf("ArgsToHex err: %s", err.Error())
		return
	}
	accountInfo := dao.TableAccountInfo{
		BlockNumber:        req.BlockNumber,
		Outpoint:           common.OutPoint2String(req.TxHash, 0),
		AccountId:          accBuilder.AccountId,
		OwnerChainType:     ownerHex.ChainType,
		Owner:              ownerHex.AddressHex,
		OwnerAlgorithmId:   ownerHex.DasAlgorithmId,
		ManagerChainType:   managerHex.ChainType,
		Manager:            managerHex.AddressHex,
		ManagerAlgorithmId: managerHex.DasAlgorithmId,
		Status:             status,
	}
	transactionInfo := dao.TableTransactionInfo{
		BlockNumber:    req.BlockNumber,
		AccountId:      accBuilder.AccountId,
		Account:        accBuilder.Account,
		Action:         req.Action,
		ServiceType:    dao.ServiceTypeRegister,
		ChainType:      ownerHex.ChainType,
		Address:        ownerHex.AddressHex,
		Capacity:       0,
		Outpoint:       common.OutPoint2String(req.TxHash, 0),
		BlockTimestamp: req.BlockTimestamp,
	}

	if err = b.dbDao.AccountCrossChain(accountInfo, transactionInfo); err != nil {
		log.Error("AccountCrossChain err:", err.Error(), req.TxHash, req.BlockNumber)
		resp.Err = fmt.Errorf("AccountCrossChain err: %s ", err.Error())
		return
	}

	return
}