package dao

import (
	"gorm.io/gorm/clause"
	"time"
)

type TableBlockInfo struct {
	Id          uint64    `json:"id" gorm:"column:id;primaryKey;type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT ''"`
	BlockNumber uint64    `json:"block_number" gorm:"column:block_number;uniqueIndex:uk_block_number;type:bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT ''"`
	BlockHash   string    `json:"block_hash" gorm:"column:block_hash;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	ParentHash  string    `json:"parent_hash" gorm:"column:parent_hash;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT ''"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT ''"`
}

const (
	TableNameBlockInfo = "t_block_info"
)

func (t *TableBlockInfo) TableName() string {
	return TableNameBlockInfo
}

func (d *DbDao) CreateBlockInfo(blockNumber uint64, blockHash, parentHash string) error {
	return d.db.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{"block_hash", "parent_hash"}),
	}).Create(&TableBlockInfo{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		ParentHash:  parentHash,
	}).Error
}

func (d *DbDao) DeleteBlockInfo(blockNumber uint64) error {
	return d.db.Where("block_number < ?", blockNumber).Delete(&TableBlockInfo{}).Error
}

func (d *DbDao) FindBlockInfo() (blockInfo TableBlockInfo, err error) {
	err = d.db.Order("block_number DESC").Limit(1).Find(&blockInfo).Error
	return
}

func (d *DbDao) FindBlockInfoByBlockNumber(blockNumber uint64) (blockInfo TableBlockInfo, err error) {
	err = d.db.Where("block_number = ?", blockNumber).Limit(1).Find(&blockInfo).Error
	return
}
