package txn

import (
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	Debit    = "DR"
	Credit   = "CR"
	BatchTXN = "ST"
	BatchPAY = "SP"
)

var (
	ErrInsufficientRecords  = errors.New("txn: Not enough records (minimum 2 required)")
	ErrInsufficientBatches  = errors.New("txn: Not enough batches (minimum 1 required)")
	ErrInvalidRecord        = errors.New("txn: Invalid Record can't be written")
	ErrBadFileHeader        = errors.New("txn: Bad File Header prevented reading")
	ErrBadBatchHeader       = errors.New("txn: Bad Batch Header prevented reading")
	ErrBadRecord            = errors.New("txn: Bad Record prevented reading")
	ErrBadBatchTrailer      = errors.New("txn: Bad Batch Trailer prevented reading")
	ErrBadFileTrailer       = errors.New("txn: Bad File Trailer prevented reading")
	ErrUnexpectedRecordType = errors.New("txn: Unexpected record type, can decode 0,1 and 7 only")

	bsbNumberRegEx = regexp.MustCompile(`^\d{3}-\d{3}$`)
)

func padRight(str, pad string, length int) string {
	for {
		str += pad
		if len(str) > length {
			return str[0:length]
		}
	}
}

func spaces(howMany int) string {
	return padRight("", " ", howMany)
}

// FileHeader TXN file header
type FileHeader struct {
	recordType     int       // pos 1      - always zero
	CustomerNumber string    // pos 1-10   - left justified e.g. 00123456
	CustomerName   string    // pos 10-45  - left justified and blank filled. e.g. AAA LEGAL SERVICES
	RemitterName   string    // pos 45-64  - left justified and blank filled. e.g. ‘MACQUARIE BANK
	FileCreated    time.Time // pos 64-72  - YYYYMMDD and zero filled
	ProcessingDate time.Time // pos 72-80  - YYYYMMDD and zero filled
	Description    string    // pos 80-100 - left justified and blank filled. e.g. ACCOUNT TRANSACTIONS or DEFT PAYMENTS
	// Space filled from 100-170. Spaces between every gap for a total 170 characters
}

func (h *FileHeader) Read(l string) error {
	if len(l) != 171 && len(l) != 172 { // 170 + '\n' || 170 + '\r\n'
		log.Println("TXN: Header expected 170, got", len(l))
		return ErrBadFileHeader
	}
	// Just read it all back in and unpack
	h.recordType, _ = strconv.Atoi(strings.TrimSpace(l[0:1]))
	h.CustomerNumber = strings.TrimSpace(l[1:10])
	h.CustomerName = strings.TrimSpace(l[10:45])
	h.RemitterName = strings.TrimSpace(l[45:64])
	h.FileCreated, _ = time.Parse("20060102", strings.TrimSpace(l[64:72]))
	h.ProcessingDate, _ = time.Parse("20060102", strings.TrimSpace(l[72:80]))
	h.Description = strings.TrimSpace(l[80:100])
	return nil
}

// BatchHeader TXN batch header per batch, multiple batches possible
type BatchHeader struct {
	recordType      int             // pos 1       - always one
	BSBNumber       string          // pos 1-8     - in the format 182-222
	AccountNumber   string          // pos 8-17    - e.g. 116217011
	AccountName     string          // pos 17-52   - left justified and blank filled. e.g. ‘DEMO ACCOUNT NUMBER 1’
	TransactionDate time.Time       // pos 52-60   - YYYYMMDD and zero filled
	Amount          decimal.Decimal // pos 60-76   - Right justified and blank filled. e.g. 123456.78
	Indicator       string          // pos 76-78   - Debit/Credit - DR or CR
	// Space filled from 78-170. Spaces between every gap for a total 170 characters
}

func (h *BatchHeader) Read(l string) error {
	if len(l) != 171 && len(l) != 172 { // 170 + '\n' || 170 + '\r\n'
		log.Println("TXN: Header expected 170, got", len(l))
		return ErrBadBatchHeader
	}
	// Just read it all back in and unpack
	h.BSBNumber = strings.TrimSpace(l[1:8])
	h.AccountNumber = strings.TrimSpace(l[8:17])
	h.AccountName = strings.TrimSpace(l[17:52])
	h.TransactionDate, _ = time.Parse("20060102", strings.TrimSpace(l[52:60]))
	h.Amount, _ = decimal.NewFromString(strings.TrimSpace(l[60:76]))
	h.Indicator = strings.TrimSpace(l[76:78])
	return nil
}

// Record ..
type Record struct {
	recordType               int             // pos 1       - always two
	BSBNumber                string          // pos 1-8     - in the format 182-222
	AccountNumber            string          // pos 8-17    - e.g. 116217011
	AccountName              string          // pos 17-52   - left justified and blank filled. e.g. ‘DEMO ACCOUNT NUMBER 1’
	TransactionDate          time.Time       // pos 52-60   - YYYYMMDD and zero filled
	Amount                   decimal.Decimal // pos 60-76   - Right justified and blank filled. e.g. 123456.78
	Indicator                string          // pos 76-78   - Debit/Credit - DR or CR
	TransactionCode          string          // pos 78-80   - Either 13, debit or 50, credit.
	Description              string          // pos 80-120  - left justified and blank filled.
	ReferenceNumber          int             // pos 120-130 - left justified and blank filled.
	SecondaryReferenceNumber string          // pos 130-140 - not utilised for General products
	ChequeNumber             string          // pos 140-148 - left justified and blank filled
	// Space filled from 148-168. Spaces between every gap for a total 168 characters
}

// IsValid performs some basic checks on records
func (r *Record) IsValid() bool {
	// Transaction validation
	switch r.Indicator {
	case Credit:
		fallthrough
	case Debit:
		// All good - next checks
	default:
		return false
	}

	// BSB validation
	return bsbNumberRegEx.MatchString(r.BSBNumber)
}

func (r *Record) Read(l string) error {
	if len(l) != 169 && len(l) != 170 { // 168 + '\n' || 168 + '\r\n'
		return ErrBadRecord
	}
	r.recordType, _ = strconv.Atoi(strings.TrimSpace(l[0:1]))
	// Just read it all back in and unpack
	r.BSBNumber = strings.TrimSpace(l[1:8])
	r.AccountNumber = strings.TrimSpace(l[8:17])
	r.AccountName = strings.TrimSpace(l[17:52])
	r.TransactionDate, _ = time.Parse("20060102", strings.TrimSpace(l[52:60]))
	r.Amount, _ = decimal.NewFromString(strings.TrimSpace(l[60:76]))
	r.Indicator = strings.TrimSpace(l[76:78])
	r.TransactionCode = strings.TrimSpace(l[78:80])
	r.Description = strings.TrimSpace(l[80:120])
	r.ReferenceNumber, _ = strconv.Atoi(strings.TrimSpace(l[120:130]))
	r.SecondaryReferenceNumber = strings.TrimSpace(l[130:140])
	r.ChequeNumber = strings.TrimSpace(l[140:148])

	if !r.IsValid() {
		return ErrInvalidRecord
	}
	return nil
}

// FileTrailer in TXN file
type FileTrailer struct {
	recordType              int             // pos 1      - always nine
	CustomerNumber          string          // pos 1-9   - left justified e.g. 00123456
	CustomerName            string          // pos 9-44  - left justified and blank filled. e.g. AAA LEGAL SERVICES
	TotalDebitTransactions  int             // pos 44-50 - Right justified and blank filled. Total number of debits in file.
	TotalCreditTransactions int             // pos 50-56 - Right justified and blank filled. Total number of credits in file.
	TotalDebitAmount        decimal.Decimal // pos 56-72 - Right justified and blank filled. Total value of debits in file.
	TotalCreditAmount       decimal.Decimal // pos 72-88 - Right justified and blank filled. Total value of credits in file.
	// Space filled from 88-170. Spaces between every gap for a total 170 characters
}

func (t *FileTrailer) Read(l string) error {
	if len(l) != 171 && len(l) != 172 { // 170 + '\n' || 170 + '\r\n'
		log.Println("TXN: Trailer expected 171, got", len(l))
		return ErrBadFileTrailer
	}
	// Just read it all back in and unpack
	t.recordType, _ = strconv.Atoi(strings.TrimSpace(l[0:1]))

	t.CustomerNumber = strings.TrimSpace(l[1:9])
	t.CustomerName = strings.TrimSpace(l[9:44])

	t.TotalDebitTransactions, _ = strconv.Atoi(strings.TrimSpace(l[44:50]))
	t.TotalCreditTransactions, _ = strconv.Atoi(strings.TrimSpace(l[50:56]))

	t.TotalDebitAmount, _ = decimal.NewFromString(strings.TrimSpace(l[56:72]))
	t.TotalCreditAmount, _ = decimal.NewFromString(strings.TrimSpace(l[72:88]))

	return nil
}

// BatchTrailer TXN batch trailer per batch, multiple batches possible
type BatchTrailer struct {
	recordType              int             // pos 1       - always two
	BSBNumber               string          // pos 1-8     - in the format 182-222
	AccountNumber           string          // pos 8-17    - e.g. 116217011
	AccountName             string          // pos 17-52   - left justified and blank filled. e.g. ‘DEMO ACCOUNT NUMBER 1’
	TransactionDate         time.Time       // pos 52-60   - YYYYMMDD and zero filled
	Amount                  decimal.Decimal // pos 60-76   - Right justified and blank filled. e.g. 123456.78
	Indicator               string          // pos 76-78   - Debit/Credit - DR or CR
	BatchType               string          // pos 78-80   - Either ST, txn or SP, pay.
	ReferenceNumber         int             // pos 80-86 - left justified and blank filled.
	TotalDebitTransactions  int             // pos 86-92 - Right justified and blank filled. Total number of debits in file.
	TotalCreditTransactions int             // pos 92-98 - Right justified and blank filled. Total number of credits in file.
	TotalDebitAmount        decimal.Decimal // pos 98-114 - Right justified and blank filled. Total value of debits in file.
	TotalCreditAmount       decimal.Decimal // pos 114-130 - Right justified and blank filled. Total value of credits in file.
	// Space filled from 130-170. Spaces between every gap for a total 170 characters
}

func (t *BatchTrailer) Read(l string) error {
	if len(l) != 171 && len(l) != 172 { // 170 + '\n' || 170 + '\r\n'
		log.Println("TXN: Batch Trailer expected 170, got", len(l))
		return ErrBadBatchTrailer
	}
	// Just read it all back in and unpack
	t.recordType, _ = strconv.Atoi(strings.TrimSpace(l[0:1]))

	t.BSBNumber = strings.TrimSpace(l[1:8])
	t.AccountNumber = strings.TrimSpace(l[8:17])
	t.AccountName = strings.TrimSpace(l[17:52])
	t.TransactionDate, _ = time.Parse("20060102", strings.TrimSpace(l[52:60]))
	t.Amount, _ = decimal.NewFromString(strings.TrimSpace(l[60:76]))
	t.Indicator = strings.TrimSpace(l[76:78])
	t.BatchType = strings.TrimSpace(l[78:80])
	t.ReferenceNumber, _ = strconv.Atoi(strings.TrimSpace(l[80:86]))

	t.TotalDebitTransactions, _ = strconv.Atoi(strings.TrimSpace(l[86:92]))
	t.TotalCreditTransactions, _ = strconv.Atoi(strings.TrimSpace(l[92:98]))

	t.TotalDebitAmount, _ = decimal.NewFromString(strings.TrimSpace(l[98:114]))
	t.TotalCreditAmount, _ = decimal.NewFromString(strings.TrimSpace(l[114:130]))

	return nil
}

func (t *BatchTrailer) Write(w io.Writer) {
	tempStr := fmt.Sprintf(
		"%d%7.7s%9.9s%-35.35s%8.8s%16.16s%2s%2s%06.6d%6.1d%6.1d%16.16s%16.16s%s",
		t.recordType,
		t.BSBNumber,
		t.AccountNumber,
		t.AccountName,
		t.TransactionDate.Format("20060102"),
		t.Amount.StringFixedBank(2),
		t.Indicator,
		t.BatchType,
		t.ReferenceNumber,
		t.TotalDebitTransactions,
		t.TotalCreditTransactions,
		t.TotalDebitAmount.StringFixedBank(2),
		t.TotalCreditAmount.StringFixedBank(2),
		spaces(40),
	)
	// Add final padding
	fmt.Fprintf(w, "%s", padRight(tempStr, " ", 170))
}

func (t *FileTrailer) Write(w io.Writer) {
	tempStr := fmt.Sprintf(
		"%d%08.8s%-35.35s%-6.1d%-6.1d%-16.16s%-16.16s%s",
		t.recordType,
		t.CustomerNumber,
		t.CustomerName,
		t.TotalDebitTransactions,
		t.TotalCreditTransactions,
		t.TotalDebitAmount.StringFixedBank(2),
		t.TotalCreditAmount.StringFixedBank(2),
		spaces(82),
	)
	// Add final padding
	fmt.Fprintf(w, "%s", padRight(tempStr, " ", 170))
}

// Write FileHeader to io.Writer
func (h *FileHeader) Write(w io.Writer) {
	tempStr := fmt.Sprintf(
		"%d%08.8s%-35.35s%-20.20s%8.8s%8.8s%20.20s%s",
		h.recordType,
		h.CustomerNumber,
		h.CustomerName,
		h.RemitterName,
		h.FileCreated.Format("20060102"),
		h.ProcessingDate.Format("20060102"),
		h.Description,
		spaces(70),
	)
	// Add final padding
	fmt.Fprintf(w, "%s", padRight(tempStr, " ", 170))
}

// Write BatchHeader to io.Writer
func (h *BatchHeader) Write(w io.Writer) {
	tempStr := fmt.Sprintf(
		"1%-7.7s%-9.9s%-35.35s%8.8s%16.16s%2.2s%s",
		h.BSBNumber,
		h.AccountNumber,
		h.AccountName,
		h.TransactionDate.Format("20060102"),
		h.Amount.StringFixedBank(2),
		h.Indicator,
		spaces(92),
	)
	// Add final padding
	fmt.Fprintf(w, "%s", padRight(tempStr, " ", 170))
}

func (r *Record) Write(w io.Writer) {
	tempStr := fmt.Sprintf(
		"2%7.7s%9.9s%-35.35s%8.8s%16.16s%2.2s%2.2s%-40.40s%-10.1d%-10.10s%-8.8s%s",
		r.BSBNumber,
		r.AccountNumber,
		r.AccountName,
		r.TransactionDate.Format("20060102"),
		r.Amount.StringFixedBank(2),
		r.Indicator,
		r.TransactionCode,
		r.Description,
		r.ReferenceNumber,
		r.SecondaryReferenceNumber,
		r.ChequeNumber,
		spaces(20),
	)
	// Add final padding
	fmt.Fprintf(w, "%s", padRight(tempStr, " ", 168))
}
