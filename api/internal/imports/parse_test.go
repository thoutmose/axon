package imports

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func TestTodoist(t *testing.T) {
	csv := "TYPE,CONTENT,DESCRIPTION,PRIORITY,INDENT,AUTHOR,RESPONSIBLE,DATE,DATE_LANG,TIMEZONE\n" +
		"section,Errands,,,,,,,,\n" +
		"task,Call the bank @phone,About the card,4,1,Ada,,2026-10-14,en,Europe/Paris\n" +
		"task,Water plants,,1,1,Ada,,every monday,en,Europe/Paris\n" +
		"note,a comment,,,,,,,,\n"
	got, err := Todoist([]byte(csv))
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if got[0].Title != "Call the bank" || got[0].Context != "@phone" || got[0].Due == nil || got[0].Due.Format("2006-01-02") != "2026-10-14" {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].Due != nil || !strings.Contains(got[1].Notes, "every monday") {
		t.Errorf("second = %+v", got[1])
	}
	if _, err := Todoist([]byte("a,b\n1,2\n")); err == nil {
		t.Error("accepted a CSV that is not from Todoist")
	}
}

func TestTickTick(t *testing.T) {
	csv := "\"Date: 2026-10-12+0000\"\n\"Version: 7.1\"\n\"Status: \n0 Normal\n1 Completed\n2 Archived\"\n" +
		"\"Folder Name\",\"List Name\",\"Title\",\"Kind\",\"Tags\",\"Content\",\"Is Check list\",\"Start Date\",\"Due Date\",\"Reminder\",\"Repeat\",\"Priority\",\"Status\"\n" +
		"\"\",\"Inbox\",\"Renew passport\",\"TEXT\",\"errands, admin\",\"Bring photos\",\"N\",\"\",\"2026-10-20T00:00:00+0000\",\"\",\"\",\"0\",\"0\"\n" +
		"\"\",\"Inbox\",\"Old thing\",\"TEXT\",\"\",\"\",\"N\",\"\",\"\",\"\",\"\",\"0\",\"2\"\n"
	got, err := TickTick([]byte(csv))
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if got[0].Context != "@errands" || got[0].Due == nil || got[0].Done || got[0].Notes != "Bring photos" {
		t.Errorf("first = %+v", got[0])
	}
	if !got[1].Done {
		t.Errorf("an archived task should import as done: %+v", got[1])
	}
}

func TestGoodreads(t *testing.T) {
	csv := "Book Id,Title,Author,Author l-f,Additional Authors,ISBN,ISBN13,My Rating,Average Rating,Publisher,Binding,Number of Pages,Year Published,Original Publication Year,Date Read,Date Added,Bookshelves,Bookshelves with positions,Exclusive Shelf\n" +
		"1,Atomic Habits,James Clear,\"Clear, James\",,=\"0735211299\",,5,4.4,Avery,Hardcover,320,2018,2018,2024/03/02,2024/01/01,,,read\n" +
		"2,Moonwalking with Einstein,Joshua Foer,,,,,0,4.0,,,,2011,2011,,2025/01/01,,,to-read\n" +
		"3,Make It Stick,Peter C. Brown,,,,,0,4.1,,,,2014,2014,,2025/02/01,,,currently-reading\n"
	got, err := Goodreads([]byte(csv))
	if err != nil || len(got) != 3 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if g := got[0]; g.Status != "done" || g.Rating != 10 || g.Year != 2018 || g.Finished != "2024-03-02" || g.Creators != "James Clear" {
		t.Errorf("first = %+v", g)
	}
	if got[1].Status != "want" || got[2].Status != "doing" || got[1].Rating != 0 {
		t.Errorf("shelves = %+v", got)
	}
}

func TestLetterboxd(t *testing.T) {
	data := zipOf(t, map[string]string{
		"watched.csv":   "Date,Name,Year,Letterboxd URI\n2024-01-05,Arrival,2016,https://boxd.it/a\n2024-02-01,Dune,2021,https://boxd.it/b\n",
		"ratings.csv":   "Date,Name,Year,Letterboxd URI,Rating\n2024-01-05,Arrival,2016,https://boxd.it/a,4.5\n",
		"diary.csv":     "Date,Name,Year,Letterboxd URI,Rating,Rewatch,Tags,Watched Date\n2024-01-06,Arrival,2016,https://boxd.it/a,4.5,,,2024-01-05\n",
		"watchlist.csv": "Date,Name,Year,Letterboxd URI\n2024-03-01,Past Lives,2023,https://boxd.it/c\n2024-03-01,Dune,2021,https://boxd.it/b\n",
	})
	got, err := Letterboxd(data)
	if err != nil || len(got) != 3 {
		t.Fatalf("got %+v, %v", got, err)
	}
	by := map[string]Media{}
	for _, m := range got {
		by[m.Title] = m
	}
	if a := by["Arrival"]; a.Rating != 9 || a.Status != "done" || a.Finished != "2024-01-05" || a.Year != 2016 {
		t.Errorf("arrival = %+v", a)
	}
	if by["Past Lives"].Status != "want" || by["Dune"].Status != "done" {
		t.Errorf("statuses = %+v", by)
	}
}

func TestMoney(t *testing.T) {
	firefly := "user_id,group_id,journal_id,created_at,updated_at,group_title,type,amount,foreign_amount,currency_code,foreign_currency_code,description,date,source_name,source_iban,source_type,destination_name,destination_iban,destination_type,reconciled,category\n" +
		"1,1,1,,,,Withdrawal,-12.50,,EUR,,Lunch,2026-10-01T12:00:00+02:00,Checking,,Asset account,Bistro,,Expense account,false,Food\n" +
		"1,2,2,,,,Deposit,2000.00,,EUR,,Salary,2026-10-01T00:00:00+02:00,Employer,,Revenue account,Checking,,Asset account,false,\n" +
		"1,3,3,,,,Transfer,500.00,,EUR,,Savings,2026-10-02T00:00:00+02:00,Checking,,Asset account,Savings,,Asset account,false,\n"
	rows, err := Firefly([]byte(firefly))
	if err != nil || len(rows) != 2 || rows[0].Cents != -1250 || rows[0].Payee != "Bistro" || rows[0].Note != "Lunch · Food" || rows[1].Cents != 200000 || rows[1].Payee != "Employer" {
		t.Fatalf("firefly = %+v, %v", rows, err)
	}
	ofx := `OFXHEADER:100
DATA:OFXSGML
<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS><BANKTRANLIST>
<STMTTRN><TRNTYPE>DEBIT<DTPOSTED>20261003120000[+2:CEST]<TRNAMT>-42.10<FITID>1<NAME>SUPERMARKET<MEMO>Card 1234
</STMTTRN>
<STMTTRN><TRNTYPE>CREDIT<DTPOSTED>20261005<TRNAMT>15.00<FITID>2<NAME>Refund
</STMTTRN>
</BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`
	rows, err = OFX([]byte(ofx))
	if err != nil || len(rows) != 2 || rows[0].On != "2026-10-03" || rows[0].Cents != -4210 || rows[0].Payee != "SUPERMARKET" || rows[0].Note != "Card 1234" || rows[1].Cents != 1500 {
		t.Fatalf("ofx = %+v, %v", rows, err)
	}
	if _, err := OFX([]byte("not ofx")); err == nil {
		t.Error("accepted a file that is not OFX")
	}
}

func TestNotion(t *testing.T) {
	data := zipOf(t, map[string]string{
		"Export-1/Garden 0123456789abcdef0123456789abcdef.md":                                          "# Garden\n\nSee [Compost](Garden%200123456789abcdef0123456789abcdef/Compost%20fedcba9876543210fedcba9876543210.md).\n\n![](Garden%200123456789abcdef0123456789abcdef/plan.png)\n",
		"Export-1/Garden 0123456789abcdef0123456789abcdef/Compost fedcba9876543210fedcba9876543210.md": "# Compost\n\nGreens and browns.\n",
		"Export-1/Garden 0123456789abcdef0123456789abcdef/plan.png":                                    "\x89PNG",
		"Export-1/Tasks 11112222333344445555666677778888.csv":                                          "Name,Done\n",
	})
	notes, files, err := Notion(data)
	if err != nil || len(notes) != 2 || len(files) != 1 {
		t.Fatalf("notes %+v files %+v err %v", notes, files, err)
	}
	by := map[string]string{}
	for _, n := range notes {
		by[n.Path] = n.Content
	}
	garden := by["Notion/Export-1/Garden.md"]
	if !strings.Contains(garden, "[[Compost]]") || !strings.Contains(garden, "![](Garden/plan.png)") {
		t.Errorf("garden = %q (paths %v)", garden, by)
	}
	if files[0].Path != "Notion/Export-1/Garden/plan.png" {
		t.Errorf("picture at %s", files[0].Path)
	}
}
