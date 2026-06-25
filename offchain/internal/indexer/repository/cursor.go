// internal/indexer/repository/cursor.go
package repository

import (
	"context"

	"gorm.io/gorm"
)

// GetCursor 获取索引进度
func (r *Repository) GetCursor(ctx context.Context, contractAddr string, chainID int64) (uint64, error) {
	var cursor SyncCursor
	err := r.db.WithContext(ctx).
		Where("contract_address = ? AND chain_id = ?", contractAddr, chainID).
		First(&cursor).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return cursor.LastBlock, nil
}

// SaveCursor 保存索引进度
func (r *Repository) SaveCursor(ctx context.Context, contractAddr string, chainID int64, lastBlock uint64) error {
	cursor := SyncCursor{
		ContractAddress: contractAddr,
		ChainID:         chainID,
		LastBlock:       lastBlock,
	}
	return r.db.WithContext(ctx).
		Where(SyncCursor{ContractAddress: contractAddr, ChainID: chainID}).
		Assign(SyncCursor{LastBlock: lastBlock}).
		FirstOrCreate(&cursor).Error
}
