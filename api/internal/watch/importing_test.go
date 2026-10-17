package watch

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseCSVReadsGoogleTakeoutSubscriptions(t *testing.T) {
	in := "Channel Id,Channel Url,Channel Title\n" +
		"UCZCFT11CWBi3MHNlGf019nw,http://www.youtube.com/channel/UCZCFT11CWBi3MHNlGf019nw,Abdul Bari\n"
	got, err := ParseCSV(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []Planned{{URL: "https://www.youtube.com/feeds/videos.xml?channel_id=UCZCFT11CWBi3MHNlGf019nw", Title: "Abdul Bari"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseYouTubeJSONMakesFoldersTopics(t *testing.T) {
	in := `{"dsa":["UCZCFT11CWBi3MHNlGf019nw","UCD8yeTczadqdARzQUp29PJw"],
		"computer_science":["UCD8yeTczadqdARzQUp29PJw","not-a-channel"],
		"ysc_deck":[],"ysc_settings":{"position":1},"ysc_meta":{"dsa":{"img":"x"}}}`
	got, err := ParseYouTubeJSON(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	feed := "https://www.youtube.com/feeds/videos.xml?channel_id="
	want := []Planned{
		{URL: feed + "UCD8yeTczadqdARzQUp29PJw", Topics: []string{"computer science", "dsa"}},
		{URL: feed + "UCZCFT11CWBi3MHNlGf019nw", Topics: []string{"dsa"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for _, bad := range []string{"[]", `{"ysc_deck":[]}`, "not json"} {
		if _, err := ParseYouTubeJSON(strings.NewReader(bad)); err == nil {
			t.Fatalf("%q: want an error", bad)
		}
	}
}
