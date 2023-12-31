package service

import (
	"context"
	"encoding/json"
	"fmt"

	// "log"
	"time"

	"go-wallet.in/domain"
	"go-wallet.in/dto"
	"go-wallet.in/internal/util"
)

type TransactionService struct {
	accountRepo      domain.AccountRepository
	transactionRepo  domain.TransactionRepository
	cacheRepo        domain.CacheRepository
	notificationRepo domain.NotificationRepository
	hub              *dto.Hub
}

func NewTransaction(accountRepo domain.AccountRepository, transactionRepo domain.TransactionRepository,
	cacheRepo domain.CacheRepository, notificationRepo domain.NotificationRepository, hub *dto.Hub) domain.TransactionService {
	return &TransactionService{
		accountRepo:      accountRepo,
		transactionRepo:  transactionRepo,
		cacheRepo:        cacheRepo,
		notificationRepo: notificationRepo,
		hub:              hub,
	}
}

func (t TransactionService) TransferInquiry(ctx context.Context, req dto.TransferInquiryReq) (dto.TransferInquiryRes, error) {
	user := ctx.Value("x-user").(dto.UserData)
	myAccount, err := t.accountRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return dto.TransferInquiryRes{}, err
	}

	if myAccount == (domain.Account{}) {
		return dto.TransferInquiryRes{}, domain.ErrAccountNotFound
	}

	dofAccount, err := t.accountRepo.FindByAccountNumber(ctx, req.AccountNumber)
	if err != nil {
		return dto.TransferInquiryRes{}, err
	}

	if dofAccount == (domain.Account{}) {
		return dto.TransferInquiryRes{}, domain.ErrAccountNotFound
	}

	if myAccount.Balance < req.Amount {
		return dto.TransferInquiryRes{}, domain.ErrInsufficientBalance
	}

	InqueryKey := util.GenerateRandomString(32)
	jsonData, _ := json.Marshal(req)
	_ = t.cacheRepo.Set(InqueryKey, jsonData)

	return dto.TransferInquiryRes{
		InquiryKey: InqueryKey,
	}, nil
}

func (t TransactionService) TransferExecute(ctx context.Context, req dto.TransferExecuteReq) error {
	val, err := t.cacheRepo.Get(req.InquiryKey)
	if err != nil {
		return domain.ErrInquiryNotFound
	}

	var reqInquiry dto.TransferInquiryReq
	_ = json.Unmarshal(val, &reqInquiry)

	if reqInquiry == (dto.TransferInquiryReq{}) {
		return domain.ErrInquiryNotFound
	}

	user := ctx.Value("x-user").(dto.UserData)
	myAccount, err := t.accountRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return err
	}

	if myAccount == (domain.Account{}) {
		return domain.ErrAccountNotFound
	}

	dofAccount, err := t.accountRepo.FindByAccountNumber(ctx, reqInquiry.AccountNumber)
	if err != nil {
		return err
	}

	if dofAccount == (domain.Account{}) {
		return domain.ErrAccountNotFound
	}

	debitTransactionDomain := domain.Transaction{
		AccountId:       myAccount.ID,
		SoftNumber:      myAccount.AccountNumber,
		DofNumber:       dofAccount.AccountNumber,
		TransactionType: "D",
		Amount:          reqInquiry.Amount,
		TransactionDate: time.Now(),
	}

	err = t.transactionRepo.Insert(ctx, &debitTransactionDomain)
	if err != nil {
		return err
	}

	creditTransactionDomain := domain.Transaction{
		AccountId:       dofAccount.ID,
		SoftNumber:      myAccount.AccountNumber,
		DofNumber:       dofAccount.AccountNumber,
		TransactionType: "C",
		Amount:          reqInquiry.Amount,
		TransactionDate: time.Now(),
	}

	err = t.transactionRepo.Insert(ctx, &creditTransactionDomain)
	if err != nil {
		return err
	}

	myAccount.Balance = myAccount.Balance - reqInquiry.Amount
	err = t.accountRepo.UpdateAccountBalance(ctx, &myAccount)
	if err != nil {
		return err
	}

	dofAccount.Balance = dofAccount.Balance + reqInquiry.Amount
	err = t.accountRepo.UpdateAccountBalance(ctx, &dofAccount)
	if err != nil {
		return err
	}

	// log.Printf("Transfer from %s to %s with amount %f success", myAccount.AccountNumber, dofAccount.AccountNumber, reqInquiry.Amount)
	go t.NotificationAfterTransfer(myAccount, dofAccount, reqInquiry.Amount)
	return nil
}

func (t TransactionService) NotificationAfterTransfer(sofAccount domain.Account, dofAccount domain.Account, amount float64) {
	notificationSender := domain.Notification{
		UserID:    sofAccount.UserId,
		Title:     "Transfer Berhasil",
		Body:      fmt.Sprintf("Anda telah melakukan transfer senilai %.2f ke %s", amount, dofAccount.AccountNumber),
		IsRead:    0,
		Status:    1,
		CreatedAt: time.Now(),
	}

	notificationReceiver := domain.Notification{
		UserID:    dofAccount.UserId,
		Title:     "Dana Diterima",
		Body:      fmt.Sprintf("Anda menerima transfer senilai %.2f dari %s", amount, sofAccount.AccountNumber),
		IsRead:    0,
		Status:    1,
		CreatedAt: time.Now(),
	}

	// log.Println("Notification Sender: ", notificationSender)
	_ = t.notificationRepo.Insert(context.Background(), &notificationSender)
	// if err != nil {
	// 	log.Println(err)
	// }
	if channel, ok := t.hub.NotificationChannel[sofAccount.UserId]; ok {
		channel <- dto.NotificationData{
			ID:        notificationSender.ID,
			Title:     notificationSender.Title,
			Body:      notificationSender.Body,
			Status:    notificationSender.Status,
			IsRead:    notificationSender.IsRead,
			CreatedAt: notificationSender.CreatedAt,
		}
	}
	// log.Println("Notification Receiver: ", notificationReceiver)
	_ = t.notificationRepo.Insert(context.Background(), &notificationReceiver)
	// if err != nil {
	// 	log.Println(err)
	// }
	if channel, ok := t.hub.NotificationChannel[dofAccount.UserId]; ok {
		channel <- dto.NotificationData{
			ID:        notificationReceiver.ID,
			Title:     notificationReceiver.Title,
			Body:      notificationReceiver.Body,
			Status:    notificationReceiver.Status,
			IsRead:    notificationReceiver.IsRead,
			CreatedAt: notificationReceiver.CreatedAt,
		}
	}
}
