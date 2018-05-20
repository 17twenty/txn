PACKAGE DOCUMENTATION

package txn
    import "github.com/17twenty/txn"


CONSTANTS

const (
    Debit    = "DR"
    Credit   = "CR"
    BatchTXN = "ST"
    BatchPAY = "SP"
)

VARIABLES

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
)

TYPES

type Batch struct {
    BatchHeader  BatchHeader
    Records      []Record
    BatchTrailer BatchTrailer
}
    Batch describes a TXN batch, a file can have multiple batches

func NewBatch() Batch
    NewBatch ..

type BatchHeader struct {
    BSBNumber       string          // pos 1-8     - in the format 182-222
    AccountNumber   string          // pos 8-17    - e.g. 116217011
    AccountName     string          // pos 17-52   - left justified and blank filled. e.g. ‘DEMO ACCOUNT NUMBER 1’
    TransactionDate time.Time       // pos 52-60   - YYYYMMDD and zero filled
    Amount          decimal.Decimal // pos 60-76   - Right justified and blank filled. e.g. 123456.78
    Indicator       string          // pos 76-78   - Debit/Credit - DR or CR
    // contains filtered or unexported fields
}
    BatchHeader TXN batch header per batch, multiple batches possible

func (h *BatchHeader) Read(l string) error

func (h *BatchHeader) Write(w io.Writer)
    Write BatchHeader to io.Writer

type BatchTrailer struct {
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
    // contains filtered or unexported fields
}
    BatchTrailer TXN batch trailer per batch, multiple batches possible

func (t *BatchTrailer) Read(l string) error

func (t *BatchTrailer) Write(w io.Writer)

type FileHeader struct {
    CustomerNumber string    // pos 1-10   - left justified e.g. 00123456
    CustomerName   string    // pos 10-45  - left justified and blank filled. e.g. AAA LEGAL SERVICES
    RemitterName   string    // pos 45-64  - left justified and blank filled. e.g. ‘MACQUARIE BANK
    FileCreated    time.Time // pos 64-72  - YYYYMMDD and zero filled
    ProcessingDate time.Time // pos 72-80  - YYYYMMDD and zero filled
    Description    string    // pos 80-100 - left justified and blank filled. e.g. ACCOUNT TRANSACTIONS or DEFT PAYMENTS
    // contains filtered or unexported fields
}
    FileHeader TXN file header

func (h *FileHeader) Read(l string) error

func (h *FileHeader) Write(w io.Writer)
    Write FileHeader to io.Writer

type FileTrailer struct {
    CustomerNumber          string          // pos 1-9   - left justified e.g. 00123456
    CustomerName            string          // pos 9-44  - left justified and blank filled. e.g. AAA LEGAL SERVICES
    TotalDebitTransactions  int             // pos 44-50 - Right justified and blank filled. Total number of debits in file.
    TotalCreditTransactions int             // pos 50-56 - Right justified and blank filled. Total number of credits in file.
    TotalDebitAmount        decimal.Decimal // pos 56-72 - Right justified and blank filled. Total value of debits in file.
    TotalCreditAmount       decimal.Decimal // pos 72-88 - Right justified and blank filled. Total value of credits in file.
    // contains filtered or unexported fields
}
    FileTrailer in TXN file

func (t *FileTrailer) Read(l string) error

func (t *FileTrailer) Write(w io.Writer)

type Reader struct {
    FileHeader  FileHeader
    Batch       []Batch
    FileTrailer FileTrailer
    // contains filtered or unexported fields
}
    A Reader reads records from an TXN file.

    As returned by NewReader, a Reader expects input conforming to spec. The
    Header and Trailer fields expose details about the underlying item

func NewReader(r io.Reader) *Reader
    NewReader returns a new Reader that reads from r.

func (r *Reader) ReadAll() (batch []Batch, err error)
    ReadAll reads all the remaining records from r.

type Record struct {
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
    // contains filtered or unexported fields
}
    Record ..

func (r *Record) IsValid() bool
    IsValid performs some basic checks on records

func (r *Record) Read(l string) error

func (r *Record) Write(w io.Writer)

type Writer struct {
    // OmitBatchTotals can be used for banks that don't summarise
    // the credit/debit transactions
    OmitBatchTotals bool
    // CRLFLineEndings allows you to toggle whether to use Windows/DOS style
    // line endings vs the default unix style
    CRLFLineEndings bool
    FileHeader      *FileHeader
    FileTrailer     *FileTrailer
    Batch           []Batch
    // contains filtered or unexported fields
}
    Writer implements buffering for an io.Writer object. If an error occurs
    writing to a Writer, no more data will be accepted and all subsequent
    writes, and Flush, will return the error. After all data has been
    written, the client should call the Flush method to guarantee all data
    has been forwarded to the underlying io.Writer.

func NewWriter(w io.Writer) *Writer
    NewWriter returns a new Writer whose buffer has the default size.

func (w *Writer) Error() error
    Error reports any error that has occurred during a previous Write or
    Flush.

func (w *Writer) Flush()
    Flush can be called to ensure all data has been written

func (w *Writer) Write() (err error)
    Write writes the entire file containing an array of Batches, each one
    with 1 or more records It returns an error if something is wrong with
    the batches/records.


