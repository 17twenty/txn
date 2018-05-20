package txn

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestLocalReader(t *testing.T) {
	f, err := os.Open("./Test_TXN_20170123.txn")

	if err != nil {
		t.Fatal("Couldn't find local test file")
	}

	txn := NewReader(f)
	batch, err := txn.ReadAll()

	if err != nil {
		t.Fatal("Expected '", nil, "' but got", err)
	}

	totalRecords := 0
	for _, v := range batch {
		totalRecords += len(v.Records)
	}

	if totalRecords != 10 || txn.FileTrailer.TotalCreditTransactions != 5 || txn.FileTrailer.TotalDebitTransactions != 5 {
		t.Fatalf("Failure - expected 10 records but got %v and %v\n", totalRecords, txn.FileTrailer.TotalCreditTransactions)
	}

	for idx, val := range []string{
		"DDR GL481         Tower Australia",
		"TEST TRANS        SIMPSON DESERT O",
		"TEST TRANS               payment",
		"PAYMENT 1246      ATHM",
		"TEST TRANS        SIMPSON DESERT O",
		"TEST TRANS        SIMPSON DESERT O",
		"TRANSA            Blue Sky Stallio",
		"Capitalised Interest",
		"Service Charge",
		"TEST ACCOUNT PAYMENT FOR     Loan to HCA",
	} {
		if val != txn.Batch[0].Records[idx].Description {
			t.Fatal("Expected '", val, "' but got '", txn.Batch[0].Records[idx].Description, "'")
		}
	}
}

func TestDemo(t *testing.T) {
	records := []Record{
		{
			AccountNumber:   "123456789",
			BSBNumber:       "182-222",
			AccountName:     "DEMO ACCOUNT NUMBER 2",
			Indicator:       Debit,
			TransactionCode: "13",
			TransactionDate: time.Now(),
			Description:     "DDR GL481         Tower Australia",
			ReferenceNumber: 245397,
			Amount:          decimal.NewFromFloat(2721.78),
		},
		{
			AccountNumber:   "123456789",
			BSBNumber:       "182-222",
			AccountName:     "DEMO ACCOUNT NUMBER 2",
			Indicator:       Credit,
			TransactionCode: "50",
			Description:     "TEST TRANS        SIMPSON DESERT O",
			Amount:          decimal.NewFromFloat(1210.00),
		},
		{
			AccountNumber:   "123456789",
			BSBNumber:       "182-222",
			AccountName:     "DEMO ACCOUNT NUMBER 2",
			Indicator:       Debit,
			TransactionCode: "13",
			Description:     "TEST TRANS               payment",
			ReferenceNumber: 0,
			Amount:          decimal.NewFromFloat(120.00),
		},
		{
			AccountNumber:   "123456789",
			BSBNumber:       "182-222",
			AccountName:     "DEMO ACCOUNT NUMBER 2",
			Indicator:       Credit,
			TransactionCode: "50",
			Description:     "PAYMENT 1246      ATHM",
			ReferenceNumber: 1,
			Amount:          decimal.NewFromFloat(2448.96),
		},
	}

	totalCreditAmount := records[1].Amount.Add(records[3].Amount)
	totalDebitAmount := records[0].Amount.Add(records[2].Amount)

	var buf bytes.Buffer
	w := NewWriter(&buf)

	w.FileHeader.CustomerNumber = "123456"
	w.FileHeader.CustomerName = "ABC PTY LIMITED"
	w.FileHeader.RemitterName = "MACQUARIE BANK"
	w.Batch[0].Records = records

	w.Batch[0].BatchHeader.BSBNumber = "182-222"
	w.Batch[0].BatchHeader.AccountNumber = "123456789"
	w.Batch[0].BatchHeader.AccountName = "DEMO ACCOUNT NUMBER 2"
	w.Batch[0].BatchHeader.TransactionDate = time.Now()
	w.Batch[0].BatchHeader.Amount = decimal.NewFromFloat(426.32)
	w.Batch[0].BatchHeader.Indicator = Credit

	w.FileTrailer.CustomerNumber = w.FileHeader.CustomerNumber
	w.FileTrailer.CustomerName = w.FileHeader.CustomerName

	if err := w.Write(); err != nil {
		t.Fatal("error writing record", err)
	}

	// Write any buffered data to the underlying writer (standard output).
	w.Flush()

	if err := w.Error(); err != nil {
		log.Println(err)
	}
	ff := NewReader(&buf)
	rr, err := ff.ReadAll()

	if err != nil {
		t.Fatal("Expected '", nil, "' but got", err)
	}
	tmp := ff.Batch[0].BatchTrailer.TotalCreditAmount.Sub(ff.Batch[0].BatchTrailer.TotalDebitAmount)

	expectedBatchTrailerBalance := tmp.Abs()
	if len(rr) != 1 {
		t.Fatalf("Failure - expected 1 batch but got %v\n", len(rr))
	}
	if len(rr[0].Records) != 4 {
		t.Fatalf("Failure - expected 4 records but got %v\n", len(rr[0].Records))
	}
	if expectedBatchTrailerBalance.Cmp(ff.Batch[0].BatchTrailer.Amount) != 0 {
		t.Fatalf("Failure - expected batch trailer amount to be %v but got %v\n", expectedBatchTrailerBalance, ff.Batch[0].BatchTrailer.Amount)
	}
	if ff.FileTrailer.TotalCreditAmount.Cmp(totalCreditAmount) != 0 {
		t.Fatalf("Failure - expected file trailer total credit amount to be %+v but got %+v\n", totalCreditAmount, ff.FileTrailer.TotalCreditAmount)
	}
	if ff.FileTrailer.TotalDebitAmount.Cmp(totalDebitAmount) != 0 {
		t.Fatalf("Failure - expected file trailer total debit amount to be %v but got %v\n", totalDebitAmount, ff.FileTrailer.TotalDebitAmount)
	}
	if ff.FileTrailer.TotalCreditTransactions != 2 {
		t.Fatalf("Failure - expected 2 total credit tx but got %v\n", ff.FileTrailer.TotalCreditTransactions)
	}
	if ff.FileTrailer.TotalDebitTransactions != 2 {
		t.Fatalf("Failure - expected 2 total debit tx but got %v\n", ff.FileTrailer.TotalDebitTransactions)
	}
}
