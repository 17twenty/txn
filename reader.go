package txn

import (
	"bufio"
	"io"
	"log"
)

// A Reader reads records from an TXN file.
//
// As returned by NewReader, a Reader expects input conforming to spec.
// The Header and Trailer fields expose details about the underlying item
type Reader struct {
	FileHeader  FileHeader
	Batch       []Batch
	FileTrailer FileTrailer
	r           *bufio.Reader
}

// Batch describes a TXN batch, a file can have multiple batches
type Batch struct {
	BatchHeader  BatchHeader
	Records      []Record
	BatchTrailer BatchTrailer
}

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: bufio.NewReader(r),
	}
}

// ReadAll reads all the remaining records from r.
func (r *Reader) ReadAll() (batch []Batch, err error) {
	for {
		err = r.readRecordOrHeaderOrTrailer()
		if err == io.EOF {
			err = nil // ReadAll is happy - not erroneous
			return r.Batch, err
		}
		if err != nil {
			log.Println("readRecordOrHeaderOrTrailer", err)
			break
		}
	}
	return r.Batch, err
}

func (r *Reader) readRecordOrHeaderOrTrailer() error {
	var (
		record Record
		batch  Batch
	)
	b, err := r.r.ReadByte()
	if err != nil || r.r.UnreadByte() != nil {
		return err
	}

	// We'll always want a line
	line, err := r.r.ReadString('\n')
	if err != nil && err != io.EOF {
		// Could be a trailer - there's no newline there. Look for EOF?
		log.Println("Didn't get a line")
		return err
	}

	switch b {
	case '0':
		err = r.FileHeader.Read(line)
	case '1':
		if err = batch.BatchHeader.Read(line); err == nil {
			r.Batch = append(r.Batch, batch)
		}
	case '2':
		err = record.Read(line)
		// No point appending garbage
		if err == nil {
			if record.IsValid() {
				r.Batch[len(r.Batch)-1].Records = append(r.Batch[len(r.Batch)-1].Records, record)
			} else {
				err = ErrInvalidRecord
			}
		}
	case '7':
		if err = batch.BatchTrailer.Read(line); err == nil {
			r.Batch[len(r.Batch)-1].BatchTrailer = batch.BatchTrailer
		}
	case '9':
		err = r.FileTrailer.Read(line)
	default:
		err = ErrUnexpectedRecordType
	}

	return err
}
