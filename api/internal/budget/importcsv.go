package budget

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
)

// ParseCents reads a signed amount as written in bank exports: "-12.50",
// "1 234,56", "1.234,56", "(12.00)", "€ 5". The last separator followed by
// one or two digits is the decimal mark. No floating point is involved.
func ParseCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg, s = true, s[1:len(s)-1]
	}
	s = strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' || r == '-' || r == '+' {
			return r
		}
		return -1 // currency symbols, spaces, apostrophes
	}, s)
	if strings.HasPrefix(s, "-") {
		neg, s = !neg, s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	} else if strings.HasSuffix(s, "-") { // some banks write "12.50-"
		neg, s = !neg, s[:len(s)-1]
	}
	if s == "" {
		return 0, errors.New("empty amount")
	}
	whole, frac := s, ""
	if i := strings.LastIndexAny(s, ".,"); i >= 0 && len(s)-i-1 >= 1 && len(s)-i-1 <= 2 {
		whole, frac = s[:i], s[i+1:]
	}
	whole = strings.NewReplacer(".", "", ",", "").Replace(whole)
	if whole == "" {
		whole = "0"
	}
	if len(whole) > 10 || strings.ContainsAny(whole+frac, "+-") {
		return 0, fmt.Errorf("not an amount: %q", s)
	}
	units, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("not an amount: %q", s)
	}
	cents := int64(0)
	if frac != "" {
		f := frac
		if len(f) == 1 {
			f += "0"
		}
		if cents, err = strconv.ParseInt(f, 10, 64); err != nil {
			return 0, fmt.Errorf("not an amount: %q", s)
		}
	}
	total := units*100 + cents
	if neg {
		total = -total
	}
	return total, nil
}

var dateLayouts = map[string]string{"YYYY-MM-DD": "2006-01-02", "DD/MM/YYYY": "02/01/2006", "MM/DD/YYYY": "01/02/2006", "DD.MM.YYYY": "02.01.2006"}

// ImportSpec says how to read a bank's CSV export. Columns are 0-based.
type ImportSpec struct {
	CSV        string     `json:"csv"`
	AccountID  *uuid.UUID `json:"account_id"`
	Delimiter  string     `json:"delimiter"`   // "," by default
	DateFormat string     `json:"date_format"` // YYYY-MM-DD by default
	HasHeader  bool       `json:"has_header"`
	Date       int        `json:"date_column"`
	Amount     int        `json:"amount_column"` // negative = expense, positive = income
	Payee      *int       `json:"payee_column"`
	Note       *int       `json:"note_column"`
}

// ImportResult says what an import did.
type ImportResult struct {
	Created  int      `json:"created"`
	Skipped  int      `json:"skipped"` // already imported
	Problems []string `json:"problems"`
}

// maxImportRows bounds one import.
const maxImportRows = 5000

// Import records each row of a CSV export once. A row's identity is a hash
// of the account, date, amount, payee, note, and how many identical rows
// came before it in the file, so re-importing the same or an overlapping
// export creates nothing twice. Rules categorise imported rows.
func Import(ctx context.Context, q db.Querier, userID uuid.UUID, spec ImportSpec) (ImportResult, error) {
	res := ImportResult{Problems: []string{}}
	layout, ok := dateLayouts[spec.DateFormat]
	if spec.DateFormat == "" {
		layout, ok = time.DateOnly, true
	}
	if !ok {
		return res, ValidationError{"date_format is YYYY-MM-DD, DD/MM/YYYY, MM/DD/YYYY, or DD.MM.YYYY."}
	}
	if spec.AccountID != nil {
		if _, err := GetAccount(ctx, q, userID, *spec.AccountID); errors.Is(err, ErrNotFound) {
			return res, ValidationError{"No such account."}
		} else if err != nil {
			return res, err
		}
	}
	r := csv.NewReader(strings.NewReader(spec.CSV))
	r.FieldsPerRecord = -1
	if spec.Delimiter != "" {
		d := []rune(spec.Delimiter)
		if len(d) != 1 {
			return res, ValidationError{"The delimiter is one character, like , or ;."}
		}
		r.Comma = d[0]
	}
	account := "none"
	if spec.AccountID != nil {
		account = spec.AccountID.String()
	}
	seen := map[string]int{}
	for line := 1; ; line++ {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return res, ValidationError{fmt.Sprintf("Line %d is not valid CSV.", line)}
		}
		if line == 1 && spec.HasHeader {
			continue
		}
		if line > maxImportRows+1 {
			return res, ValidationError{fmt.Sprintf("Import at most %d rows at a time.", maxImportRows)}
		}
		col := func(i *int) string {
			if i == nil || *i < 0 || *i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[*i])
		}
		dateCol, amountCol := spec.Date, spec.Amount
		day, err := time.Parse(layout, col(&dateCol))
		if err != nil {
			res.Problems = append(res.Problems, fmt.Sprintf("Line %d: the date %q does not match %s.", line, col(&dateCol), spec.DateFormat))
			continue
		}
		cents, err := ParseCents(col(&amountCol))
		if err != nil || cents == 0 || cents > MaxCents || cents < -MaxCents {
			res.Problems = append(res.Problems, fmt.Sprintf("Line %d: %q is not an amount.", line, col(&amountCol)))
			continue
		}
		kind := "income"
		if cents < 0 {
			kind, cents = "expense", -cents
		}
		on, payee, note := day.Format(time.DateOnly), col(spec.Payee), col(spec.Note)
		if len([]rune(payee)) > 200 {
			payee = string([]rune(payee)[:200])
		}
		key := strings.Join([]string{account, on, kind, strconv.FormatInt(cents, 10), payee, note}, "\x1f")
		sum := sha256.Sum256([]byte(key + "\x1f" + strconv.Itoa(seen[key])))
		seen[key]++
		hash := hex.EncodeToString(sum[:])
		f := TransactionFields{OccurredOn: &on, Kind: &kind, AmountCents: &cents, AccountID: spec.AccountID, Payee: &payee, Note: &note}
		switch _, err := createTransaction(ctx, q, userID, f, nil, &hash); {
		case err == nil:
			res.Created++
		case errors.Is(err, errDuplicate):
			res.Skipped++
		default:
			var v ValidationError
			if errors.As(err, &v) {
				res.Problems = append(res.Problems, fmt.Sprintf("Line %d: %s", line, v.Msg))
				continue
			}
			return res, err
		}
	}
	return res, nil
}
