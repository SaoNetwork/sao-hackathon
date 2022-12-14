package model

import (
	"fmt"
	"github.com/shopspring/decimal"
	"path/filepath"
)

type UserProfile struct {
	SaoModel
	EthAddr  string
	Avatar   string
	Username string
}

type UserProfileVO struct {
	Avatar   string
	Username string
}

type UserSummary struct {
	SpaceUsed     int64
	SpaceQuota    int64
	Applications  int
	TotalUploads  int
	PublicUploads int
	Collections   int
	PurchaseSummary
	SellSummary
}

type UserDashboard struct {
	RecentUploads []FileInfoInMarket
	TotalUploads  int64
}

type UserPurchases struct {
	Purchases      []FileInfoInMarket
	TotalPurchases int64
}

type PurchaseSummary struct {
	PurchasesFiles int
	TotalPaid      decimal.Decimal
}

type SellSummary struct {
	SellFiles   int
	TotalEarned decimal.Decimal
}

func (model *Model) UpsertUserProfile(ethAddr string, updateProfile UserProfile) (*UserProfile, error) {
	condition := UserProfile{EthAddr: ethAddr}
	var user UserProfile
	result := model.DB.Where(condition).Assign(updateProfile).FirstOrCreate(&user, condition)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (model *Model) UpdateUsername(ethAddr string, username string) error {
	condition := UserProfile{EthAddr: ethAddr}
	result := model.DB.Where(condition).Update("username", username)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (model *Model) GetUserProfile(ethAddr string) (*UserProfile, error) {
	var user UserProfile
	user.EthAddr = ethAddr
	user.Username = fmt.Sprintf("%s_%s", "Storverse", ethAddr[len(ethAddr)-4:])
	model.DB.Where(&UserProfile{EthAddr: ethAddr}).FirstOrCreate(&user)
	return &user, nil
}

func (model *Model) GetUserSummary(ethAddr string) (*UserSummary, error) {
	var uploads int64
	condition := &FilePreview{EthAddr: ethAddr}
	result := model.DB.Model(&FilePreview{}).Where(condition).Where("status = 1 or (status = 2 and price = 0) or (status = 2 and price > 0 and nft_token_id > 0)").Count(&uploads)

	var sellSummary SellSummary
	model.DB.Table("file_previews").Select("sum(price) as total_earned, count(*) as sell_files").
		Joins("inner join purchase_orders on file_previews.id = purchase_orders.file_id").Where("file_previews.eth_addr = ?", ethAddr).Scan(&sellSummary)

	var purchaseSummary PurchaseSummary
	model.DB.Model(&FilePreview{}).Select("sum(price) as total_paid, count(*) as purchases_files").
		Joins("inner join purchase_orders on file_previews.id = purchase_orders.file_id").Where("purchase_orders.buyer_addr = ?", ethAddr).Scan(&purchaseSummary)

	userSummary := UserSummary{
		Applications:  5,
		PublicUploads: int(uploads),
		TotalUploads:  int(uploads),
		PurchaseSummary: PurchaseSummary{
			TotalPaid:      purchaseSummary.TotalPaid,
			PurchasesFiles: purchaseSummary.PurchasesFiles,
		},
		SellSummary: SellSummary{
			SellFiles:   sellSummary.SellFiles,
			TotalEarned: sellSummary.TotalEarned,
		},
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &userSummary, nil
}

func (model *Model) GetUserDashboard(limit int, offset int, ethAddr string, previewPath func(string) string) (*UserDashboard, error) {
	dashboard := UserDashboard{}

	// recent uploads
	var uploads []FilePreview
	condition := &FilePreview{EthAddr: ethAddr}
	result := model.DB.Model(&FilePreview{}).Where(condition).Where("status = 1 or (status = 2 and price = 0) or (status = 2 and price > 0 and nft_token_id > 0)").Order("created_at desc").Limit(limit).Offset(offset).Find(&uploads)
	if result.Error != nil {
		return nil, result.Error
	}

	var fileInfoInMarket []FileInfoInMarket
	for _, upload := range uploads {
		fileExtension := filepath.Ext(upload.Filename)
		if fileExtension != "" {
			fileExtension = fileExtension[1:]
		}
		fileInfoInMarket = append(fileInfoInMarket, FileInfoInMarket{Id: upload.Id,
			CreatedAt:      upload.CreatedAt,
			UpdatedAt:      upload.UpdatedAt,
			EthAddr:        upload.EthAddr,
			Preview:        previewPath(upload.Preview),
			Labels:         upload.Labels,
			Price:          upload.Price,
			Title:          upload.Title,
			Description:    upload.Description,
			ContentType:    upload.ContentType,
			Type:           upload.Type,
			Status:         upload.Status,
			NftTokenId:     upload.NftTokenId,
			FileCategory:   upload.FileCategory,
			AdditionalInfo: upload.AdditionalInfo,
			FileExtension:  fileExtension,
			AlreadyPaid:    true})
	}
	dashboard.RecentUploads = fileInfoInMarket

	// total uploads
	var totalUploads int64
	result = model.DB.Model(&FilePreview{}).Where(condition).Where("status = 1 or (status = 2 and price = 0) or (status = 2 and price > 0 and nft_token_id > 0)").Count(&totalUploads)
	if result.Error != nil {
		return nil, result.Error
	}
	dashboard.TotalUploads = totalUploads

	return &dashboard, nil
}

func (model *Model) GetUserPurchases(limit int, offset int, ethAddr string, previewPath func(string) string) (*UserPurchases, error) {
	purchases := UserPurchases{}

	// recent uploads
	var uploads []FilePreview
	result := model.DB.Model(&FilePreview{}).Joins("RIGHT JOIN purchase_orders ON purchase_orders.file_id = file_previews.id").Where("purchase_orders.buyer_addr = ?", ethAddr).Limit(limit).Offset(offset).Order("created_at desc").Find(&uploads)
	if result.Error != nil {
		return nil, result.Error
	}

	var fileInfoInMarket []FileInfoInMarket
	for _, upload := range uploads {
		fileInfoInMarket = append(fileInfoInMarket, FileInfoInMarket{Id: upload.Id,
			CreatedAt:      upload.CreatedAt,
			UpdatedAt:      upload.UpdatedAt,
			EthAddr:        upload.EthAddr,
			Preview:        previewPath(upload.Preview),
			Labels:         upload.Labels,
			Price:          upload.Price,
			Title:          upload.Title,
			Description:    upload.Description,
			ContentType:    upload.ContentType,
			Type:           upload.Type,
			Status:         upload.Status,
			NftTokenId:     upload.NftTokenId,
			FileCategory:   upload.FileCategory,
			AdditionalInfo: upload.AdditionalInfo,
			AlreadyPaid:    true})
	}
	purchases.Purchases = fileInfoInMarket

	// total uploads
	var totalPurchases int64
	result = model.DB.Model(&FilePreview{}).Joins("RIGHT JOIN purchase_orders ON purchase_orders.file_id = file_previews.id").Where("purchase_orders.buyer_addr = ?", ethAddr).Count(&totalPurchases)
	if result.Error != nil {
		return nil, result.Error
	}
	purchases.TotalPurchases = totalPurchases

	return &purchases, nil
}
