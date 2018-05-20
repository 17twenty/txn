package txn

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/shopspring/decimal"
)

// Writer implements buffering for an io.Writer object.
// If an error occurs writing to a Writer, no more data will be
// accepted and all subsequent writes, and Flush, will return the error.
// After all data has been written, the client should call the
// Flush method to guarantee all data has been forwarded to
// the underlying io.Writer.
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
	wr              *bufio.Writer
}

// NewWriter returns a new Writer whose buffer has the default size.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		wr: bufio.NewWriter(w),
		FileHeader: &FileHeader{
			recordType:     0,
			FileCreated:    time.Now(),
			ProcessingDate: time.Now(),
			Description:    "ACCOUNT TRANSACTIONS",
		},
		Batch: []Batch{
			NewBatch(),
		},

		FileTrailer: &FileTrailer{
			recordType: 9,
		},
	}
}

// NewBatch ..
func NewBatch() Batch {
	return Batch{
		BatchHeader: BatchHeader{
			recordType:      1,
			TransactionDate: time.Now(),
		},
		BatchTrailer: BatchTrailer{
			recordType:      7,
			TransactionDate: time.Now(),
			BatchType:       BatchPAY,
		},
	}
}

// Write writes the entire file containing an array of Batches, each one with 1 or more records
// It returns an error if something is wrong with the batches/records.
func (w *Writer) Write() (err error) {
	if len(w.Batch) < 1 {
		return ErrInsufficientBatches
	}

	w.FileHeader.Write(w.wr)
	if w.CRLFLineEndings {
		w.wr.WriteByte('\r')
	}
	w.wr.WriteByte('\n')

	for k, batch := range w.Batch {
		batch.BatchHeader.Write(w.wr)
		if w.CRLFLineEndings {
			w.wr.WriteByte('\r')
		}
		var batchDebitCounter int
		var batchCreditCounter int
		var batchDebitTx decimal.Decimal
		var batchCreditTx decimal.Decimal
		w.wr.WriteByte('\n')

		for i, r := range batch.Records {
			// Validation spin...
			if !r.IsValid() {
				return fmt.Errorf("%v (record %d)", ErrInvalidRecord, i)
			}
			if !w.OmitBatchTotals {
				switch r.Indicator {
				case Debit:
					w.FileTrailer.TotalDebitAmount = w.FileTrailer.TotalDebitAmount.Add(r.Amount)
					w.FileTrailer.TotalDebitTransactions++
					batchDebitCounter++
					batchDebitTx = batchDebitTx.Add(r.Amount)
				case Credit:
					w.FileTrailer.TotalCreditAmount = w.FileTrailer.TotalCreditAmount.Add(r.Amount)
					w.FileTrailer.TotalCreditTransactions++
					batchCreditCounter++
					batchCreditTx = batchCreditTx.Add(r.Amount)

				default:
					log.Println("Unknown transaction type", r.Indicator, "in record", i)
				}
			}

			r.Write(w.wr)

			if w.CRLFLineEndings {
				w.wr.WriteByte('\r')
			}
			w.wr.WriteByte('\n')
		}
		batchAmount := batchCreditTx.Sub(batchDebitTx)
		indicator := "CR"
		if batchAmount.Sign() < 0 {
			indicator = "DR"
		}
		batch.BatchTrailer.BSBNumber = batch.BatchHeader.BSBNumber
		batch.BatchTrailer.AccountNumber = batch.BatchHeader.AccountNumber
		batch.BatchTrailer.AccountName = batch.BatchHeader.AccountName
		batch.BatchTrailer.TransactionDate = time.Now()
		batch.BatchTrailer.Amount = batchAmount.Abs()
		batch.BatchTrailer.Indicator = indicator
		batch.BatchTrailer.BatchType = BatchTXN
		batch.BatchTrailer.ReferenceNumber = k
		batch.BatchTrailer.TotalDebitTransactions = batchDebitCounter
		batch.BatchTrailer.TotalCreditTransactions = batchCreditCounter
		batch.BatchTrailer.TotalDebitAmount = batchDebitTx
		batch.BatchTrailer.TotalCreditAmount = batchCreditTx

		batch.BatchTrailer.Write(w.wr)
		if w.CRLFLineEndings {
			w.wr.WriteByte('\r')
		}
		w.wr.WriteByte('\n')
	}

	// Last part is to get net trailer amount
	// Some banks require a balancing line at the bottom
	// We're going to omit it unless told otherwise
	w.FileTrailer.Write(w.wr)
	if w.CRLFLineEndings {
		w.wr.WriteByte('\r')
	}
	w.wr.WriteByte('\n')
	return nil
}

// Flush can be called to ensure all data has been written
func (w *Writer) Flush() {
	w.wr.Flush()
}

// Error reports any error that has occurred during a previous Write or Flush.
func (w *Writer) Error() error {
	_, err := w.wr.Write(nil)
	return err
}
